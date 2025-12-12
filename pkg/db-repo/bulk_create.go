package db_repo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

const defaultBatchSize = 100

// bulkOperation defines the type of bulk SQL operation.
type bulkOperation string

const (
	bulkOperationInsert  bulkOperation = "INSERT INTO"
	bulkOperationReplace bulkOperation = "REPLACE INTO"
)

// BatchReplaceOption is a functional option for BatchReplace.
type BatchReplaceOption func(*batchReplaceOptions)

type batchReplaceOptions struct {
	suspendForeignKeyChecks bool
}

// WithSuspendForeignKeyChecks suspends foreign key checks before the replace
// operation and re-enables them afterward. This is useful when replacing records
// that have foreign key relationships.
func WithSuspendForeignKeyChecks() BatchReplaceOption {
	return func(opts *batchReplaceOptions) {
		opts.suspendForeignKeyChecks = true
	}
}

// BatchCreate inserts multiple records at once using a bulk INSERT statement.
// Unlike the standard gormbulk library, this implementation supports explicit IDs
// by only excluding AUTO_INCREMENT fields when they are blank (zero/nil).
// This makes it suitable for fixture loading where specific IDs are required.
//
// The batchSize parameter controls how many records are inserted per statement
// to avoid exceeding database parameter limits. Use 0 for the default batch size (100).
//
// Note: This method does not perform cross-model validation because it's designed
// for fixture loading where the model struct name may not match the table naming
// convention. The table name is always taken from the repository's metadata.
func (r *repository) BatchCreate(ctx context.Context, values []ModelBased, batchSize int) error {
	return r.doBulkOperation(ctx, values, batchSize, bulkOperationInsert, "BatchCreate", false)
}

// BatchReplace replaces multiple records at once using a bulk REPLACE INTO statement.
// Unlike BatchCreate which uses INSERT, REPLACE will delete and re-insert rows that
// would cause duplicate key violations. This is useful for upserting fixture data.
//
// The batchSize parameter controls how many records are replaced per statement
// to avoid exceeding database parameter limits. Use 0 for the default batch size (100).
//
// Options:
//   - WithSuspendForeignKeyChecks(): Suspends foreign key checks before the operation
//     and re-enables them afterward.
func (r *repository) BatchReplace(ctx context.Context, values []ModelBased, batchSize int, opts ...BatchReplaceOption) error {
	options := &batchReplaceOptions{}
	for _, opt := range opts {
		opt(options)
	}

	return r.doBulkOperation(ctx, values, batchSize, bulkOperationReplace, "BatchReplace", options.suspendForeignKeyChecks)
}

func (r *repository) doBulkOperation(ctx context.Context, values []ModelBased, batchSize int, operation bulkOperation, spanName string, suspendForeignKeyChecks bool) error {
	if len(values) == 0 {
		return nil
	}

	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	modelId := r.GetModelId()

	ctx, span := r.startSubSpan(ctx, spanName)
	defer span.Finish()

	// Set timestamps for all records
	now := r.clock.Now()
	for _, value := range values {
		value.SetUpdatedAt(&now)
		value.SetCreatedAt(&now)
	}

	// Convert to []any for bulk operation
	objects := make([]any, len(values))
	for i, v := range values {
		objects[i] = v
	}

	// Use the table name from metadata
	db := r.orm.Table(r.metadata.TableName)

	// Suspend foreign key checks if requested
	if suspendForeignKeyChecks {
		foreignKeyChecksSuspended := false
		defer func() {
			if !foreignKeyChecksSuspended {
				return
			}

			if err := db.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
				r.logger.Error(ctx, "could not re-enable foreign key checks: %w", err)
			}
		}()

		if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
			r.logger.Error(ctx, "could not suspend foreign key checks: %w", err)

			return fmt.Errorf("could not suspend foreign key checks: %w", err)
		}

		foreignKeyChecksSuspended = true
	}

	err := bulkExecute(db, objects, batchSize, operation)
	if err != nil {
		r.logger.Error(ctx, "could not %s models of type %v: %w", spanName, modelId, err)

		return err
	}

	r.logger.Info(ctx, "%s %d models of type %s", spanName, len(values), modelId)

	return nil
}

// bulkExecute executes a bulk INSERT or REPLACE statement for the given objects.
// This is a fork of github.com/t-tiger/gorm-bulk-insert that supports explicit IDs.
func bulkExecute(db *gorm.DB, objects []any, chunkSize int, operation bulkOperation, excludeColumns ...string) error {
	for _, objSet := range splitObjects(objects, chunkSize) {
		if err := executeObjSet(db, objSet, operation, excludeColumns...); err != nil {
			return err
		}
	}

	return nil
}

func executeObjSet(db *gorm.DB, objects []any, operation bulkOperation, excludeColumns ...string) error {
	if len(objects) == 0 {
		return nil
	}

	firstAttrs, err := extractMapValue(objects[0], excludeColumns)
	if err != nil {
		return err
	}

	attrSize := len(firstAttrs)

	// Scope to eventually run SQL
	mainScope := db.NewScope(objects[0])
	// Store placeholders for embedding variables
	placeholders := make([]string, 0, attrSize)

	// Replace with database column name
	dbColumns := make([]string, 0, attrSize)
	for _, key := range sortedKeys(firstAttrs) {
		dbColumns = append(dbColumns, mainScope.Quote(key))
	}

	for _, obj := range objects {
		objAttrs, err := extractMapValue(obj, excludeColumns)
		if err != nil {
			return err
		}

		// If object sizes are different, SQL statement loses consistency
		if len(objAttrs) != attrSize {
			return errors.New("attribute sizes are inconsistent")
		}

		scope := db.NewScope(obj)

		// Append variables
		variables := make([]string, 0, attrSize)
		for _, key := range sortedKeys(objAttrs) {
			scope.AddToVars(objAttrs[key])
			variables = append(variables, "?")
		}

		valueQuery := "(" + strings.Join(variables, ", ") + ")"
		placeholders = append(placeholders, valueQuery)

		// Also append variables to mainScope
		mainScope.SQLVars = append(mainScope.SQLVars, scope.SQLVars...)
	}

	insertOption := ""
	if val, ok := db.Get("gorm:insert_option"); ok {
		strVal, ok := val.(string)
		if !ok {
			return errors.New("gorm:insert_option should be a string")
		}
		insertOption = strVal
	}

	mainScope.Raw(fmt.Sprintf("%s %s (%s) VALUES %s %s",
		operation,
		mainScope.QuotedTableName(),
		strings.Join(dbColumns, ", "),
		strings.Join(placeholders, ", "),
		insertOption,
	))

	return db.Exec(mainScope.SQL, mainScope.SQLVars...).Error
}

// extractMapValue obtains columns and values required for insert from interface.
// This is a fork of the gormbulk version that supports explicit IDs:
// - AUTO_INCREMENT fields are only excluded when they are blank (IsBlank=true)
// - This allows fixtures to specify explicit IDs while still supporting auto-generated IDs
func extractMapValue(value any, excludeColumns []string) (map[string]any, error) {
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
		value = rv.Interface()
	}
	if rv.Kind() != reflect.Struct {
		return nil, errors.New("value must be kind of Struct")
	}

	attrs := map[string]any{}

	for _, field := range (&gorm.Scope{Value: value}).Fields() {
		// Exclude relational record because it's not directly contained in database columns
		_, hasForeignKey := field.TagSettingsGet("FOREIGNKEY")

		// Key difference from gormbulk: only exclude AUTO_INCREMENT fields when blank
		// This allows explicit IDs to be inserted for fixtures
		isAutoIncrementAndBlank := fieldIsAutoIncrement(field) && field.IsBlank

		if containString(excludeColumns, field.Struct.Name) ||
			field.Relationship != nil ||
			hasForeignKey ||
			field.IsIgnored ||
			isAutoIncrementAndBlank ||
			fieldIsPrimaryAndBlank(field) {
			continue
		}

		switch {
		case (field.Struct.Name == "CreatedAt" || field.Struct.Name == "UpdatedAt") && field.IsBlank:
			attrs[field.DBName] = time.Now()
		case field.HasDefaultValue && field.IsBlank:
			// If default value presents and field is empty, assign a default value
			if val, ok := field.TagSettingsGet("DEFAULT"); ok {
				attrs[field.DBName] = val
			} else {
				attrs[field.DBName] = field.Field.Interface()
			}
		default:
			attrs[field.DBName] = field.Field.Interface()
		}
	}

	return attrs, nil
}

func fieldIsAutoIncrement(field *gorm.Field) bool {
	if value, ok := field.TagSettingsGet("AUTO_INCREMENT"); ok {
		return !strings.EqualFold(value, "false")
	}

	return false
}

func fieldIsPrimaryAndBlank(field *gorm.Field) bool {
	return field.IsPrimaryKey && field.IsBlank
}

// splitObjects separates objects into chunks of specified size
func splitObjects(objArr []any, size int) [][]any {
	var chunkSet [][]any
	var chunk []any

	for len(objArr) > size {
		chunk, objArr = objArr[:size], objArr[size:]
		chunkSet = append(chunkSet, chunk)
	}
	if len(objArr) > 0 {
		chunkSet = append(chunkSet, objArr)
	}

	return chunkSet
}

// sortedKeys enables map keys to be retrieved in same order when iterating
func sortedKeys(val map[string]any) []string {
	keys := make([]string, 0, len(val))
	for key := range val {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}

// containString checks if string value is contained in slice
func containString(s []string, value string) bool {
	for _, v := range s {
		if v == value {
			return true
		}
	}

	return false
}
