package dbtx

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/db"
	db_repo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/refl"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

const (
	Create = "create"
	Read   = "read"
	Update = "update"
	Delete = "delete"
	Query  = "query"
)

var (
	operations     = []string{Create, Read, Update, Delete, Query}
	ErrCrossQuery  = fmt.Errorf("cross querying wrong model from repo")
	ErrCrossCreate = fmt.Errorf("cross creating wrong model from repo")
	ErrCrossRead   = fmt.Errorf("cross reading wrong model from repo")
	ErrCrossDelete = fmt.Errorf("cross deleting wrong model from repo")
	ErrCrossUpdate = fmt.Errorf("cross updating wrong model from repo")
)

type Entity[I comparable] interface {
	GetId() I
	GetCreatedAt() time.Time
	SetCreatedAt(createdAt time.Time)
	GetUpdatedAt() time.Time
	SetUpdatedAt(updatedAt time.Time)
}

type Repository[I comparable, E Entity[I]] interface {
	NewTx(ctx context.Context) *TxContext
	Create(tx *TxContext, entity E) error
	Read(tx *TxContext, id I) (E, error)
}

type repository[I comparable, E Entity[I]] struct {
	logger      log.Logger
	tracer      tracing.Tracer
	orm         *gorm.DB
	clock       clock.Clock
	metadata    db_repo.Metadata
	modelSource func() E
}

func New[I comparable, E Entity[I]](config cfg.Config, logger log.Logger, settings db_repo.Settings) (Repository[I, E], error) {
	tracer, err := tracing.ProvideTracer(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	orm, err := db_repo.NewOrm(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	orm.Callback().
		Update().
		After("gorm:update_time_stamp").
		Register("gosoline:ignore_created_at_if_needed", ignoreCreatedAtIfNeeded)

	return NewWithInterfaces[I, E](logger, tracer, orm, clock.Provider, settings), nil
}

func NewWithInterfaces[I comparable, E Entity[I]](logger log.Logger, tracer tracing.Tracer, orm *gorm.DB, clock clock.Clock, settings db_repo.Settings) Repository[I, E] {
	return &repository[I, E]{
		logger:      logger,
		tracer:      tracer,
		orm:         orm,
		clock:       clock,
		metadata:    settings.Metadata,
		modelSource: CreateModel[E],
	}
}

func (r *repository[I, E]) NewTx(ctx context.Context) *TxContext {
	db := r.orm.BeginTx(ctx, &sql.TxOptions{})

	return &TxContext{
		db: db,
	}
}

func (r *repository[I, E]) Create(tx *TxContext, value E) error {
	isQueryableModel := r.isQueryableModel(tx, value)

	if !isQueryableModel {
		return ErrCrossCreate
	}

	modelId := r.GetModelId()
	logger := r.logger.WithContext(tx)

	tx, span := r.startSubSpan(tx, "Create")
	defer span.Finish()

	now := r.clock.Now()
	value.SetUpdatedAt(now)
	value.SetCreatedAt(now)

	err := tx.db.Create(value).Error

	if db.IsDuplicateEntryError(err) {
		logger.Warn("could not create model of type %s due to duplicate entry error: %s", modelId, err.Error())

		return &db.DuplicateEntryError{
			Err: err,
		}
	}

	if err != nil {
		return fmt.Errorf("could not create model of type %v: %w", modelId, err)
	}

	err = r.refreshAssociations(tx, value, Create)

	if err != nil {
		return fmt.Errorf("could not update associations of model type %v: %w", modelId, err)
	}

	logger.Info("created model of type %s with id %d", modelId, value.GetId())

	created, err := r.Read(tx, value.GetId())
	if err != nil {
		return err
	}

	setValue(value, created)

	return nil
}

func (r *repository[I, E]) Read(tx *TxContext, id I) (E, error) {
	entity := r.modelSource()

	if !r.isQueryableModel(tx, entity) {
		return entity, ErrCrossRead
	}

	modelId := r.GetModelId()
	_, span := r.startSubSpan(tx, "Get")
	defer span.Finish()

	err := tx.db.First(entity, id).Error

	if gorm.IsRecordNotFoundError(err) {
		return entity, NewRecordNotFoundError(0, modelId, err)
	}

	return entity, err
}

func (r *repository[I, E]) GetModelId() string {
	return r.metadata.ModelId.String()
}

func (r *repository[I, E]) GetModelName() string {
	return r.metadata.ModelId.Name
}

func (r *repository[I, E]) startSubSpan(tx *TxContext, action string) (*TxContext, tracing.Span) {
	modelName := r.GetModelId()
	spanName := fmt.Sprintf("db_repo.%v.%v", modelName, action)

	ctx, span := r.tracer.StartSubSpan(tx, spanName)
	span.AddMetadata("model", modelName)

	tx.ctx = ctx

	return tx, span
}

func (r *repository[I, E]) isQueryableModel(tx *TxContext, model interface{}) bool {
	tableName := tx.db.NewScope(model).TableName()
	equal := strings.EqualFold(tableName, r.metadata.TableName)

	return equal || tableName == ""
}

func (r *repository[I, E]) checkResultModel(tx *TxContext, result interface{}) error {
	if refl.IsSlice(result) {
		return fmt.Errorf("result slice has to be pointer to slice")
	}

	if refl.IsPointerToSlice(result) {
		model := reflect.ValueOf(result).Elem().Interface()

		if !r.isQueryableModel(tx, model) {
			return fmt.Errorf("cross querying result slice has to be of same model")
		}
	}

	return nil
}

func (r *repository[I, E]) refreshAssociations(tx *TxContext, model interface{}, op string) error {
	typeReflection := reflect.TypeOf(model).Elem()
	valueReflection := reflect.ValueOf(model).Elem()

	for i := 0; i < typeReflection.NumField(); i++ {
		field := typeReflection.Field(i)
		tag := field.Tag.Get("orm")

		if tag == "" {
			continue
		}

		tags := make(map[string]string)
		for _, tag := range strings.Split(tag, ",") {
			parts := strings.Split(tag, ":")

			value := ""
			if len(parts) == 2 {
				value = parts[1]
			}

			tags[parts[0]] = value
		}

		if _, ok := tags["assoc_update"]; !ok {
			continue
		}

		var err error

		values := valueReflection.Field(i)
		scope := tx.db.NewScope(model)
		scopeField, _ := scope.FieldByName(field.Name)

		switch op {
		case Create, Update:
			switch scopeField.Relationship.Kind {
			case "many_to_many":
				err = tx.db.Model(model).Association(scopeField.Name).Replace(values.Interface()).Error

			default:
				assocIds := readIdsFromReflectValue(values)
				parentId := valueReflection.FieldByName("Id").Elem().Interface()

				tableName := scopeField.DBName
				if tags["assoc_update"] != "" {
					tableName = tags["assoc_update"]
				}

				qry := fmt.Sprintf("DELETE FROM %s WHERE %s = %d", tableName, scopeField.Relationship.ForeignDBNames[0], parentId)

				if len(assocIds) != 0 {
					qry += fmt.Sprintf(" AND %s NOT IN (%s)", "id", strings.Join(assocIds, ","))
				}

				err = tx.db.Exec(qry).Error
			}

		case Delete:
			switch scopeField.Relationship.Kind {
			case "has_many":
				id := valueReflection.FieldByName("Id").Elem().Interface()
				tableName := scopeField.DBName

				if tags["assoc_update"] != "" {
					tableName = tags["assoc_update"]
				}

				qry := fmt.Sprintf("DELETE FROM %s WHERE %s = %d", tableName, scopeField.Relationship.ForeignDBNames[0], id)
				err = tx.db.Exec(qry).Error

			default:
				err = tx.db.Model(model).Association(field.Name).Clear().Error
			}

		default:
			err = fmt.Errorf("unknown operation")
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func readIdsFromReflectValue(values reflect.Value) []string {
	ids := make([]string, 0)

	for j := 0; j < values.Len(); j++ {
		id := values.Index(j).Elem().FieldByName("Id").Interface().(*uint)
		ids = append(ids, strconv.Itoa(int(*id)))
	}

	return ids
}

func ignoreCreatedAtIfNeeded(scope *gorm.Scope) {
	// if you perform an update and do not specify the CreatedAt field on your data, gorm will set it to time.Time{}
	// (0000-00-00 00:00:00 in mysql). To avoid this, we mark the field as ignored if it is empty
	if m, ok := getModel(scope.Value); ok && (m.GetCreatedAt() == nil || *m.GetCreatedAt() == time.Time{}) {
		scope.Search.Omit("CreatedAt")
	}
}

func getModel(value interface{}) (db_repo.TimestampAware, bool) {
	if value == nil {
		return nil, false
	}

	if m, ok := value.(db_repo.TimestampAware); ok {
		return m, true
	}

	if val := reflect.ValueOf(value); val.Kind() == reflect.Ptr {
		return getModel(val.Elem().Interface())
	}

	return nil, false
}

func CreateModel[M any]() M {
	var model M
	value := reflect.ValueOf(model)
	valueType := value.Type()

	switch value.Kind() {
	case reflect.Pointer:
		return reflect.New(valueType.Elem()).Interface().(M)

	case reflect.Map:
		return reflect.MakeMap(valueType).Interface().(M)

	default:
		return model
	}
}

func setValue[M any](value M, target M) {
	reflected := reflect.ValueOf(value)

	if reflected.Kind() == reflect.Ptr {
		reflected.Elem().Set(reflect.ValueOf(target).Elem())
	}
}
