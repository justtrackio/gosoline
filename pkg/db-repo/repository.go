package db_repo

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
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

type Settings struct {
	cfg.AppId
	Metadata   Metadata
	ClientName string
}

//go:generate go run github.com/vektra/mockery/v2 --name RepositoryReadOnly
type RepositoryReadOnly interface {
	Read(ctx context.Context, id *uint, out ModelBased) error
	Query(ctx context.Context, qb *QueryBuilder, result any) error
	Count(ctx context.Context, qb *QueryBuilder, model ModelBased) (int, error)

	GetModelId() string
	GetModelName() string
	GetMetadata() Metadata
}

//go:generate go run github.com/vektra/mockery/v2 --name Repository
type Repository interface {
	RepositoryReadOnly
	Create(ctx context.Context, value ModelBased) error
	Update(ctx context.Context, value ModelBased) error
	Delete(ctx context.Context, value ModelBased) error
}

type repository struct {
	logger   log.Logger
	tracer   tracing.Tracer
	orm      *gorm.DB
	clock    clock.Clock
	metadata Metadata
}

func New(ctx context.Context, config cfg.Config, logger log.Logger, settings Settings) (*repository, error) {
	var err error
	var tracer tracing.Tracer
	var orm *gorm.DB

	if tracer, err = tracing.ProvideTracer(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	if orm, err = NewOrm(ctx, config, logger, settings.ClientName); err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	orm.Callback().
		Update().
		After("gorm:update_time_stamp").
		Register("gosoline:ignore_created_at_if_needed", ignoreCreatedAtIfNeeded)
	clk := clock.Provider

	return NewWithInterfaces(logger, tracer, orm, clk, settings.Metadata), nil
}

func NewWithDbSettings(ctx context.Context, config cfg.Config, logger log.Logger, dbSettings *db.Settings, repoSettings Settings) (*repository, error) {
	tracer, err := tracing.ProvideTracer(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	orm, err := NewOrmWithDbSettings(ctx, config, logger, repoSettings.ClientName, dbSettings, repoSettings.Application)
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	orm.Callback().
		Update().
		After("gorm:update_time_stamp").
		Register("gosoline:ignore_created_at_if_needed", ignoreCreatedAtIfNeeded)

	clk := clock.Provider

	return NewWithInterfaces(logger, tracer, orm, clk, repoSettings.Metadata), nil
}

func NewWithInterfaces(logger log.Logger, tracer tracing.Tracer, orm *gorm.DB, clock clock.Clock, metadata Metadata) *repository {
	return &repository{
		logger:   logger,
		tracer:   tracer,
		orm:      orm,
		clock:    clock,
		metadata: metadata,
	}
}

func (r *repository) GetOrm() *gorm.DB {
	return r.orm
}

func (r *repository) Create(ctx context.Context, value ModelBased) error {
	if !r.isQueryableModel(value) {
		return fmt.Errorf("table %q: %w", r.orm.NewScope(value).TableName(), ErrCrossCreate)
	}

	modelId := r.GetModelId()

	ctx, span := r.startSubSpan(ctx, "Create")
	defer span.Finish()

	now := r.clock.Now()
	value.SetUpdatedAt(&now)
	value.SetCreatedAt(&now)

	err := r.orm.Create(value).Error

	if db.IsDuplicateEntryError(err) {
		r.logger.Warn(ctx, "could not create model of type %s due to duplicate entry error: %s", modelId, err.Error())

		return &db.DuplicateEntryError{
			Err: err,
		}
	}

	if err != nil {
		r.logger.Error(ctx, "could not create model of type %v: %w", modelId, err)

		return err
	}

	err = r.refreshAssociations(value, Create)
	if err != nil {
		r.logger.Error(ctx, "could not update associations of model type %v: %w", modelId, err)

		return err
	}

	r.logger.Info(ctx, "created model of type %s with id %d", modelId, *value.GetId())

	return r.Read(ctx, value.GetId(), value)
}

func (r *repository) Read(ctx context.Context, id *uint, out ModelBased) error {
	if !r.isQueryableModel(out) {
		return fmt.Errorf("table %q: %w", r.orm.NewScope(out).TableName(), ErrCrossRead)
	}

	modelId := r.GetModelId()
	_, span := r.startSubSpan(ctx, "Get")
	defer span.Finish()

	err := r.orm.Unscoped().First(out, *id).Error

	if gorm.IsRecordNotFoundError(err) {
		return NewRecordNotFoundError(*id, modelId, err)
	}

	return err
}

func (r *repository) Update(ctx context.Context, value ModelBased) error {
	if !r.isQueryableModel(value) {
		return fmt.Errorf("table %q: %w", r.orm.NewScope(value).TableName(), ErrCrossUpdate)
	}

	modelId := r.GetModelId()

	ctx, span := r.startSubSpan(ctx, "UpdateItem")
	defer span.Finish()

	now := r.clock.Now()
	value.SetUpdatedAt(&now)

	err := r.orm.Save(value).Error

	if db.IsDuplicateEntryError(err) {
		r.logger.Warn(ctx, "could not update model of type %s with id %d due to duplicate entry error: %s", modelId, mdl.EmptyIfNil(value.GetId()), err.Error())

		return &db.DuplicateEntryError{
			Err: err,
		}
	}

	if err != nil {
		r.logger.Error(ctx, "could not update model of type %s with id %d: %w", modelId, mdl.EmptyIfNil(value.GetId()), err)

		return err
	}

	err = r.refreshAssociations(value, Update)
	if err != nil {
		r.logger.Error(ctx, "could not update associations of model type %s with id %d: %w", modelId, *value.GetId(), err)

		return err
	}

	r.logger.Info(ctx, "updated model of type %s with id %d", modelId, *value.GetId())

	return r.Read(ctx, value.GetId(), value)
}

func (r *repository) Delete(ctx context.Context, value ModelBased) error {
	if !r.isQueryableModel(value) {
		return fmt.Errorf("table %q: %w", r.orm.NewScope(value).TableName(), ErrCrossDelete)
	}

	modelId := r.GetModelId()

	_, span := r.startSubSpan(ctx, "Delete")
	defer span.Finish()

	err := r.refreshAssociations(value, Delete)
	if err != nil {
		r.logger.Error(ctx, "could not delete associations of model type %s with id %d: %w", modelId, *value.GetId(), err)

		return err
	}

	err = r.orm.Delete(value).Error
	if err != nil {
		r.logger.Error(ctx, "could not delete model of type %s with id %d: %w", modelId, *value.GetId(), err)
	}

	r.logger.Info(ctx, "deleted model of type %s with id %d", modelId, *value.GetId())

	return err
}

func (r *repository) isQueryableModel(model any) bool {
	tableName := r.orm.NewScope(model).TableName()

	return strings.EqualFold(tableName, r.GetMetadata().TableName) || tableName == ""
}

func (r *repository) checkResultModel(result any) error {
	if refl.IsSlice(result) {
		return fmt.Errorf("result slice has to be pointer to slice")
	}

	if refl.IsPointerToSlice(result) {
		model := reflect.ValueOf(result).Elem().Interface()

		if !r.isQueryableModel(model) {
			return fmt.Errorf("table %q: %w", r.orm.NewScope(model).TableName(), fmt.Errorf("cross querying result slice has to be of same model"))
		}
	}

	return nil
}

func (r *repository) Query(ctx context.Context, qb *QueryBuilder, result any) error {
	err := r.checkResultModel(result)
	if err != nil {
		return err
	}

	_, span := r.startSubSpan(ctx, "Query")
	defer span.Finish()

	db := r.orm.New()

	for _, j := range qb.joins {
		db = db.Joins(j)
	}

	for i := range qb.where {
		currentWhere := qb.where[i]
		if reflect.TypeOf(currentWhere).Kind() == reflect.Ptr || reflect.TypeOf(currentWhere).Kind() == reflect.Struct {
			if !r.isQueryableModel(currentWhere) {
				return fmt.Errorf("table %q: %w", r.orm.NewScope(currentWhere).TableName(), ErrCrossQuery)
			}
		}

		db = db.Where(currentWhere, qb.args[i]...)
	}

	for _, g := range qb.groupBy {
		db = db.Group(g)
	}

	for _, o := range qb.orderBy {
		db = db.Order(fmt.Sprintf("%s %s", o.field, o.direction))
	}

	if qb.page != nil {
		db = db.Offset(qb.page.offset)
		db = db.Limit(qb.page.limit)
	}

	db = db.Table(r.GetMetadata().TableName)

	err = db.Find(result).Error

	if gorm.IsRecordNotFoundError(err) {
		return NewNoQueryResultsError(r.GetModelId(), err)
	}

	return err
}

func (r *repository) Count(ctx context.Context, qb *QueryBuilder, model ModelBased) (int, error) {
	_, span := r.startSubSpan(ctx, "Count")
	defer span.Finish()

	result := struct {
		Count int
	}{}

	db := r.orm.New()

	for _, j := range qb.joins {
		db = db.Joins(j)
	}

	for i := range qb.where {
		db = db.Where(qb.where[i], qb.args[i]...)
	}

	scope := r.orm.NewScope(model)
	tableName := scope.TableName()
	key := scope.PrimaryKey()
	sel := fmt.Sprintf("COUNT(DISTINCT %s.%s) AS count", tableName, key)

	err := db.Table(tableName).Select(sel).Scan(&result).Error

	return result.Count, err
}

func (r *repository) refreshAssociations(model any, op string) error {
	typeReflection := reflect.TypeOf(model).Elem()

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

		scope := r.orm.NewScope(model)
		scopeField, _ := scope.FieldByName(field.Name)

		switch op {
		case Create, Update:
			err = r.refreshAssociationsCreate(model, i, tags, scopeField)

		case Delete:
			err = r.refreshAssociationsDelete(model, field, tags, scopeField)

		default:
			err = fmt.Errorf("unknown operation")
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) refreshAssociationsCreate(model any, fieldNum int, tags map[string]string, scopeField *gorm.Field) (err error) {
	valueReflection := reflect.ValueOf(model).Elem()
	values := valueReflection.Field(fieldNum)

	switch scopeField.Relationship.Kind {
	case "many_to_many":
		err = r.orm.Model(model).Association(scopeField.Name).Replace(values.Interface()).Error

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

		err = r.orm.Exec(qry).Error
	}

	return
}

func (r *repository) refreshAssociationsDelete(model any, field reflect.StructField, tags map[string]string, scopeField *gorm.Field) (err error) {
	valueReflection := reflect.ValueOf(model).Elem()

	switch scopeField.Relationship.Kind {
	case "has_many":
		id := valueReflection.FieldByName("Id").Elem().Interface()
		tableName := scopeField.DBName

		if tags["assoc_update"] != "" {
			tableName = tags["assoc_update"]
		}

		qry := fmt.Sprintf("DELETE FROM %s WHERE %s = %d", tableName, scopeField.Relationship.ForeignDBNames[0], id)
		err = r.orm.Exec(qry).Error

	default:
		err = r.orm.Model(model).Association(field.Name).Clear().Error
	}

	return
}

func (r *repository) GetModelId() string {
	return r.metadata.ModelId.String()
}

func (r *repository) GetModelName() string {
	return r.metadata.ModelId.Name
}

func (r *repository) GetMetadata() Metadata {
	return r.metadata
}

func (r *repository) startSubSpan(ctx context.Context, action string) (context.Context, tracing.Span) {
	modelName := r.GetModelId()
	spanName := fmt.Sprintf("db_repo.%v.%v", modelName, action)

	ctx, span := r.tracer.StartSubSpan(ctx, spanName)
	span.AddMetadata("model", modelName)

	return ctx, span
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
	if m, ok := getModel(scope.Value); ok && (m.GetCreatedAt() == nil || m.GetCreatedAt().Equal(time.Time{})) {
		scope.Search.Omit("CreatedAt")
	}
}

func getModel(value any) (TimestampAware, bool) {
	if value == nil {
		return nil, false
	}

	if m, ok := value.(TimestampAware); ok {
		return m, true
	}

	if val := reflect.ValueOf(value); val.Kind() == reflect.Ptr {
		return getModel(val.Elem().Interface())
	}

	return nil, false
}
