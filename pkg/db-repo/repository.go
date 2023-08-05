package db_repo

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
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
type RepositoryReadOnly[K mdl.PossibleIdentifier, M ModelBased[K]] interface {
	Read(ctx context.Context, id K) (M, error)
	Query(ctx context.Context, qb *QueryBuilder) ([]M, error)
	Count(ctx context.Context, qb *QueryBuilder) (int, error)

	GetModelId() string
	GetModelName() string
	GetMetadata() Metadata
}

//go:generate go run github.com/vektra/mockery/v2 --name Repository
type Repository[K mdl.PossibleIdentifier, M ModelBased[K]] interface {
	RepositoryReadOnly[K, M]
	Create(ctx context.Context, value M) error
	Update(ctx context.Context, value M) error
	Delete(ctx context.Context, value M) error
}

type ConfigurableRepository[K mdl.PossibleIdentifier, M ModelBased[K]] interface {
	Repository[K, M]
	SetModelSource(modelSource func() M)
}

type repository[K mdl.PossibleIdentifier, M ModelBased[K]] struct {
	logger      log.Logger
	tracer      tracing.Tracer
	orm         *gorm.DB
	clock       clock.Clock
	metadata    Metadata
	modelSource func() M
}

func New[K mdl.PossibleIdentifier, M ModelBased[K]](
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	settings Settings,
) (ConfigurableRepository[K, M], error) {
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
		Register("gosoline:ignore_created_at_if_needed", ignoreCreatedAtIfNeeded[K, M])
	clk := clock.Provider

	return NewWithInterfaces[K, M](logger, tracer, orm, clk, settings.Metadata, CreateModel[M]), nil
}

func NewWithDbSettings[K mdl.PossibleIdentifier, M ModelBased[K]](
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	dbSettings *db.Settings,
	repoSettings Settings,
) (ConfigurableRepository[K, M], error) {
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
		Register("gosoline:ignore_created_at_if_needed", ignoreCreatedAtIfNeeded[K, M])

	clk := clock.Provider

	return NewWithInterfaces[K, M](logger, tracer, orm, clk, repoSettings.Metadata, CreateModel[M]), nil
}

func NewWithInterfaces[K mdl.PossibleIdentifier, M ModelBased[K]](
	logger log.Logger,
	tracer tracing.Tracer,
	orm *gorm.DB,
	clock clock.Clock,
	metadata Metadata,
	modelSource func() M,
) ConfigurableRepository[K, M] {
	return &repository[K, M]{
		logger:      logger,
		tracer:      tracer,
		orm:         orm,
		clock:       clock,
		metadata:    metadata,
		modelSource: modelSource,
	}
}

func (r *repository[K, M]) GetOrm() *gorm.DB {
	return r.orm
}

func (r *repository[K, M]) SetModelSource(modelSource func() M) {
	r.modelSource = modelSource
}

func (r *repository[K, M]) Create(ctx context.Context, value M) error {
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
		return fmt.Errorf("could not create model of type %s: %w", modelId, err)
	}

	err = r.refreshAssociations(value, Create)
	if err != nil {
		return fmt.Errorf("could not update associations of model type %s: %w", modelId, err)
	}

	r.logger.Info(ctx, "created model of type %s with id %v", modelId, *value.GetId())

	created, err := r.Read(ctx, *value.GetId())
	if err != nil {
		return err
	}

	setValue(value, created)

	return nil
}

func (r *repository[K, M]) Read(ctx context.Context, id K) (M, error) {
	out := r.modelSource()

	if !r.isQueryableModel(out) {
		return out, fmt.Errorf("table %q: %w", r.orm.NewScope(out).TableName(), ErrCrossRead)
	}

	modelId := r.GetModelId()
	_, span := r.startSubSpan(ctx, "Get")
	defer span.Finish()

	err := r.orm.First(out, map[string]any{
		r.metadata.PrimaryKeyWithoutTable(): id,
	}).Error

	if gorm.IsRecordNotFoundError(err) {
		return out, NewRecordNotFoundError(idToString(id), modelId, err)
	}

	return out, err
}

func (r *repository[K, M]) Update(ctx context.Context, value M) error {
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
		r.logger.Warn(ctx, "could not update model of type %s with id %v due to duplicate entry error: %s", modelId, mdl.EmptyIfNil(value.GetId()), err.Error())

		return &db.DuplicateEntryError{
			Err: err,
		}
	}

	if err != nil {
		return fmt.Errorf("could not update model of type %s with id %v: %w", modelId, mdl.EmptyIfNil(value.GetId()), err)
	}

	err = r.refreshAssociations(value, Update)
	if err != nil {
		return fmt.Errorf("could not update associations of model type %s with id %v: %w", modelId, *value.GetId(), err)
	}

	r.logger.Info(ctx, "updated model of type %s with id %v", modelId, *value.GetId())

	updated, err := r.Read(ctx, *value.GetId())
	if err != nil {
		return err
	}

	setValue(value, updated)

	return nil
}

func (r *repository[K, M]) Delete(ctx context.Context, value M) error {
	if !r.isQueryableModel(value) {
		return fmt.Errorf("table %q: %w", r.orm.NewScope(value).TableName(), ErrCrossDelete)
	}

	modelId := r.GetModelId()

	_, span := r.startSubSpan(ctx, "Delete")
	defer span.Finish()

	err := r.refreshAssociations(value, Delete)
	if err != nil {
		return fmt.Errorf("could not delete associations of model type %s with id %v: %w", modelId, *value.GetId(), err)
	}

	err = r.orm.Delete(value).Error
	if err != nil {
		return fmt.Errorf("could not delete model of type %s with id %v: %w", modelId, *value.GetId(), err)
	}

	r.logger.Info(ctx, "deleted model of type %s with id %v", modelId, *value.GetId())

	return nil
}

func (r *repository[K, M]) isQueryableModel(model any) bool {
	tableName := r.orm.NewScope(model).TableName()

	return strings.EqualFold(tableName, r.GetMetadata().TableName) || tableName == ""
}

func (r *repository[K, M]) checkResultModel() error {
	model := r.modelSource()

	if !r.isQueryableModel(model) {
		return fmt.Errorf("table %q: %w", r.orm.NewScope(model).TableName(), fmt.Errorf("cross querying result slice has to be of same model"))
	}

	return nil
}

func (r *repository[K, M]) Query(ctx context.Context, qb *QueryBuilder) ([]M, error) {
	err := r.checkResultModel()
	if err != nil {
		return nil, err
	}

	_, span := r.startSubSpan(ctx, "Query")
	defer span.Finish()

	query := r.orm.New()

	for _, j := range qb.joins {
		query = query.Joins(j)
	}

	for i, currentWhere := range qb.where {
		if reflect.TypeOf(currentWhere).Kind() == reflect.Ptr || reflect.TypeOf(currentWhere).Kind() == reflect.Struct {
			if !r.isQueryableModel(currentWhere) {
				return nil, fmt.Errorf("table %q: %w", r.orm.NewScope(currentWhere).TableName(), ErrCrossQuery)
			}
		}

		query = query.Where(currentWhere, qb.args[i]...)
	}

	for _, g := range qb.groupBy {
		query = query.Group(g)
	}

	for _, o := range qb.orderBy {
		query = query.Order(fmt.Sprintf("%s %s", o.field, o.direction))
	}

	if qb.page != nil {
		query = query.Offset(qb.page.offset)
		query = query.Limit(qb.page.limit)
	}

	query = query.Table(r.GetMetadata().TableName)

	result := make([]M, 0)
	err = query.Find(&result).Error

	if gorm.IsRecordNotFoundError(err) {
		return nil, NewNoQueryResultsError(r.GetModelId(), err)
	}

	return result, err
}

func (r *repository[K, M]) Count(ctx context.Context, qb *QueryBuilder) (int, error) {
	_, span := r.startSubSpan(ctx, "Count")
	defer span.Finish()

	result := struct {
		Count int
	}{}

	query := r.orm.New()

	for _, j := range qb.joins {
		query = query.Joins(j)
	}

	for i, currentWhere := range qb.where {
		if reflect.TypeOf(currentWhere).Kind() == reflect.Ptr || reflect.TypeOf(currentWhere).Kind() == reflect.Struct {
			if !r.isQueryableModel(currentWhere) {
				return 0, ErrCrossQuery
			}
		}

		query = query.Where(currentWhere, qb.args[i]...)
	}

	model := r.modelSource()
	scope := r.orm.NewScope(model)
	tableName := scope.TableName()
	key := scope.PrimaryKey()
	sel := fmt.Sprintf("COUNT(DISTINCT %s.%s) AS count", tableName, key)

	err := query.Table(tableName).Select(sel).Scan(&result).Error

	return result.Count, err
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

func (r *repository[K, M]) refreshAssociations(model M, op string) error {
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

func (r *repository[K, M]) refreshAssociationsCreate(model any, fieldNum int, tags map[string]string, scopeField *gorm.Field) error {
	valueReflection := reflect.ValueOf(model).Elem()
	values := valueReflection.Field(fieldNum)

	if scopeField.Relationship.Kind == "many_to_many" {
		return r.orm.Model(model).Association(scopeField.Name).Replace(values.Interface()).Error
	}

	assocIds := readIdsFromReflectValue[K](values)
	parentId := valueReflection.FieldByName("Id").Elem().Interface()

	tableName := scopeField.DBName
	if tags["assoc_update"] != "" {
		tableName = tags["assoc_update"]
	}

	qry := fmt.Sprintf("DELETE FROM %s WHERE %s = %d", tableName, scopeField.Relationship.ForeignDBNames[0], parentId)

	if len(assocIds) != 0 {
		qry += fmt.Sprintf(" AND %s NOT IN (%s)", "id", strings.Join(assocIds, ","))
	}

	return r.orm.Exec(qry).Error
}

func (r *repository[K, M]) refreshAssociationsDelete(model any, field reflect.StructField, tags map[string]string, scopeField *gorm.Field) error {
	valueReflection := reflect.ValueOf(model).Elem()

	if scopeField.Relationship.Kind == "has_many" {
		id := valueReflection.FieldByName("Id").Elem().Interface()
		tableName := scopeField.DBName

		if tags["assoc_update"] != "" {
			tableName = tags["assoc_update"]
		}

		qry := fmt.Sprintf("DELETE FROM %s WHERE %s = %d", tableName, scopeField.Relationship.ForeignDBNames[0], id)
		return r.orm.Exec(qry).Error
	}

	return r.orm.Model(model).Association(field.Name).Clear().Error
}

func (r *repository[K, M]) GetModelId() string {
	return r.metadata.ModelId.String()
}

func (r *repository[K, M]) GetModelName() string {
	return r.metadata.ModelId.Name
}

func (r *repository[K, M]) GetMetadata() Metadata {
	return r.metadata
}

func (r *repository[K, M]) startSubSpan(ctx context.Context, action string) (context.Context, tracing.Span) {
	modelName := r.GetModelId()
	spanName := fmt.Sprintf("db_repo.%v.%v", modelName, action)

	ctx, span := r.tracer.StartSubSpan(ctx, spanName)
	span.AddMetadata("model", modelName)

	return ctx, span
}

func readIdsFromReflectValue[K mdl.PossibleIdentifier](values reflect.Value) []string {
	ids := make([]string, 0)

	for j := 0; j < values.Len(); j++ {
		id := values.Index(j).Elem().FieldByName("Id").Interface().(*K)
		ids = append(ids, idToString(*id))
	}

	return ids
}

func idToString[K mdl.PossibleIdentifier](id K) string {
	return fmt.Sprintf("%v", id)
}

func ignoreCreatedAtIfNeeded[K mdl.PossibleIdentifier, M ModelBased[K]](scope *gorm.Scope) {
	// if you perform an update and do not specify the CreatedAt field on your data, gorm will set it to time.Time{}
	// (0000-00-00 00:00:00 in mysql). To avoid this, we mark the field as ignored if it is empty

	if m, ok := scope.Value.(M); ok && mdl.IsNilOrEmpty(m.GetCreatedAt()) {
		scope.Search.Omit("CreatedAt")
	}
}
