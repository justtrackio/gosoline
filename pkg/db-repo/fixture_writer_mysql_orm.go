package db_repo

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
)

type MysqlOrmSettings struct {
	BatchSize int
}

type mysqlOrmFixtureWriter struct {
	logger    log.Logger
	metadata  *Metadata
	repo      BatchedRepository
	batchSize int
}

func MysqlOrmFixtureSetFactory[T any](metadata *Metadata, data fixtures.NamedFixtures[T], options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var writer fixtures.FixtureWriter

		if writer, err = NewMysqlOrmFixtureWriter(ctx, config, logger, metadata); err != nil {
			return nil, fmt.Errorf("failed to create mysql orm fixture writer for %s: %w", metadata.ModelId.String(), err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewMysqlOrmFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, metadata *Metadata) (fixtures.FixtureWriter, error) {
	if err := metadata.ModelId.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("can not pad model id from config: %w", err)
	}

	appId, err := cfg.GetAppIdFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("can not get app id from config: %w", err)
	}

	repoSettings := Settings{
		AppId:      appId,
		Metadata:   *metadata,
		ClientName: "default",
	}

	var dbSettings *db.Settings
	var repo *repository

	if dbSettings, err = db.ReadSettings(config, "default"); err != nil {
		return nil, fmt.Errorf("can not create repo: %w", err)
	}
	dbSettings.Parameters["FOREIGN_KEY_CHECKS"] = "0"

	if repo, err = NewWithDbSettings(ctx, config, logger, dbSettings, repoSettings); err != nil {
		return nil, fmt.Errorf("can not create repo: %w", err)
	}

	return NewMysqlFixtureWriterWithInterfaces(logger, metadata, repo, nil), nil
}

func NewMysqlFixtureWriterWithInterfaces(logger log.Logger, metadata *Metadata, repo BatchedRepository, settings *MysqlOrmSettings) fixtures.FixtureWriter {
	batchSize := fixtures.DefaultBatchSize
	if settings != nil && settings.BatchSize > 0 {
		batchSize = settings.BatchSize
	}

	return &mysqlOrmFixtureWriter{
		logger:    logger,
		metadata:  metadata,
		repo:      repo,
		batchSize: batchSize,
	}
}

func (m *mysqlOrmFixtureWriter) Write(ctx context.Context, fixtures []any) error {
	if len(fixtures) == 0 {
		return nil
	}

	// Convert fixtures to []ModelBased for BatchCreate
	models := make([]ModelBased, 0, len(fixtures))
	for _, item := range fixtures {
		model, ok := item.(ModelBased)
		if !ok {
			return fmt.Errorf("assertion failed: %T is not db_repo.ModelBased", item)
		}
		models = append(models, model)
	}

	// Check if the model has GORM relationships/associations
	// If so, we need to use individual Create calls which handle relationships properly
	if hasGormRelationships(models[0]) {
		return m.writeWithRelationships(ctx, models)
	}

	// Use BatchCreate with ON DUPLICATE KEY UPDATE for fixture loading
	// This allows fixtures to be reloaded/updated without failing on duplicates
	if err := m.repo.BatchCreate(ctx, models, m.batchSize, WithOnDuplicateKeyBehavior(DuplicateKeyUpdate)); err != nil {
		return fmt.Errorf("can not batch insert fixtures: %w", err)
	}

	m.logger.Info(ctx, "batch loaded %d mysql fixtures", len(fixtures))

	return nil
}

// writeWithRelationships inserts fixtures one by one using the repository's Create method.
// This is used when the model has GORM relationships that need to be handled properly.
func (m *mysqlOrmFixtureWriter) writeWithRelationships(ctx context.Context, models []ModelBased) error {
	for _, model := range models {
		if err := m.repo.Update(ctx, model); err != nil {
			return fmt.Errorf("can not create (update) fixture with relationships: %w", err)
		}
	}

	m.logger.Info(ctx, "loaded %d mysql fixtures", len(models))

	return nil
}

// hasGormRelationships checks if the given model has any GORM relationship fields.
// It inspects the struct fields for relationship indicators like:
// - Fields with gorm tags containing relationship keywords (foreignkey, references, many2many, etc.)
// - Fields that GORM detects as having a Relationship
func hasGormRelationships(model any) bool {
	scope := &gorm.Scope{Value: model}

	for _, field := range scope.Fields() {
		// Check if GORM detected this field as a relationship
		if field.Relationship != nil {
			return true
		}

		// Check for FOREIGNKEY tag which indicates a relationship
		if _, hasForeignKey := field.TagSettingsGet("FOREIGNKEY"); hasForeignKey {
			return true
		}

		// Check for REFERENCES tag which indicates a relationship
		if _, hasReferences := field.TagSettingsGet("REFERENCES"); hasReferences {
			return true
		}

		// Check for MANY2MANY tag which indicates a many-to-many relationship
		if _, hasMany2Many := field.TagSettingsGet("MANY2MANY"); hasMany2Many {
			return true
		}

		// Check for ASSOCIATION_FOREIGNKEY tag
		if _, hasAssocFK := field.TagSettingsGet("ASSOCIATION_FOREIGNKEY"); hasAssocFK {
			return true
		}

		// Check for JOINTABLE_FOREIGNKEY tag (many-to-many join tables)
		if _, hasJoinFK := field.TagSettingsGet("JOINTABLE_FOREIGNKEY"); hasJoinFK {
			return true
		}

		// Check for ASSOCIATION_JOINTABLE_FOREIGNKEY tag
		if _, hasAssocJoinFK := field.TagSettingsGet("ASSOCIATION_JOINTABLE_FOREIGNKEY"); hasAssocJoinFK {
			return true
		}

		// Check for POLYMORPHIC tag which indicates a polymorphic relationship
		if _, hasPolymorphic := field.TagSettingsGet("POLYMORPHIC"); hasPolymorphic {
			return true
		}
	}

	// Also check for the custom orm tag used in gosoline for association updates
	rv := reflect.ValueOf(model)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if tag := field.Tag.Get("orm"); tag != "" {
			// If there's an orm tag with assoc_update, it indicates a relationship
			if containsAssocUpdate(tag) {
				return true
			}
		}
	}

	return false
}

// containsAssocUpdate checks if the orm tag contains assoc_update
func containsAssocUpdate(tag string) bool {
	for _, part := range splitTag(tag) {
		if part == "assoc_update" || len(part) > 12 && part[:12] == "assoc_update" {
			return true
		}
	}
	return false
}

// splitTag splits a tag by comma
func splitTag(tag string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(tag); i++ {
		if tag[i] == ',' {
			parts = append(parts, tag[start:i])
			start = i + 1
		}
	}
	if start < len(tag) {
		parts = append(parts, tag[start:])
	}
	return parts
}
