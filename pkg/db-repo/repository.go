package db_repo

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"reflect"
	"strings"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/refl"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	BatchCreate = "batchCreate"
	BatchUpdate = "batchUpdate"
	BatchDelete = "batchDelete"
	Create      = "create"
	Read        = "read"
	Update      = "update"
	Delete      = "delete"
	Query       = "query"
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
	Metadata Metadata
}

//go:generate mockery --name RepositoryReadOnly
type RepositoryReadOnly interface {
	Read(ctx context.Context, id *uint, out ModelBased) error
	Query(ctx context.Context, qb *QueryBuilder, result interface{}) error
	Count(ctx context.Context, qb *QueryBuilder, model ModelBased) (int, error)

	GetModelId() string
	GetModelName() string
	GetMetadata() Metadata
}

//go:generate mockery --name Repository
type Repository interface {
	RepositoryReadOnly
	BatchCreate(ctx context.Context, values interface{}) error
	BatchUpdate(ctx context.Context, values interface{}) error
	BatchDelete(ctx context.Context, values interface{}) error
	Create(ctx context.Context, value ModelBased) error
	Delete(ctx context.Context, value ModelBased) error
	Update(ctx context.Context, value ModelBased) error
}

type repository struct {
	logger      log.Logger
	tracer      tracing.Tracer
	orm         *gorm.DB
	metadata    Metadata
	schemaCache *sync.Map
}

func New(config cfg.Config, logger log.Logger, s Settings) (*repository, error) {
	tracer, err := tracing.ProvideTracer(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	orm, err := NewOrm(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	return NewWithInterfaces(logger, tracer, orm, s.Metadata), nil
}

func NewWithDbSettings(config cfg.Config, logger log.Logger, dbSettings db.Settings, repoSettings Settings) (*repository, error) {
	tracer, err := tracing.ProvideTracer(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	orm, err := NewOrmWithDbSettings(logger, dbSettings, repoSettings.Application)
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	return NewWithInterfaces(logger, tracer, orm, repoSettings.Metadata), nil
}

func NewWithInterfaces(logger log.Logger, tracer tracing.Tracer, orm *gorm.DB, metadata Metadata) *repository {
	return &repository{
		logger:      logger,
		tracer:      tracer,
		orm:         orm,
		metadata:    metadata,
		schemaCache: &sync.Map{},
	}
}

func (r *repository) BatchCreate(ctx context.Context, values interface{}) error {
	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)
	if err != nil {
		return fmt.Errorf("could not turn values into slice: %w", err)
	}

	if len(valuesSlice) == 0 {
		return nil
	}

	queryable, err := r.isQueryableModel(valuesSlice[0])
	if err != nil {
		return err
	}

	if !queryable {
		return ErrCrossCreate
	}

	valueType := reflect.TypeOf(valuesSlice[0])
	for i := 0; i < len(valuesSlice); i++ {
		if _, ok := valuesSlice[i].(ModelBased); !ok {
			return fmt.Errorf("you should pass a slice of ModelBased, found element at %d with %T", i, valuesSlice[0])
		}

		if valueType != reflect.TypeOf(valuesSlice[i]) {
			return fmt.Errorf("your elements should have all the same types, %d was different", i)
		}
	}

	modelId := r.GetModelId()
	logger := r.logger.WithContext(ctx)

	ctx, span := r.startSubSpan(ctx, "CreateItems")
	defer span.Finish()

	orm := r.orm.
		WithContext(ctx).
		Session(&gorm.Session{
			FullSaveAssociations: true,
		})

	for _, preload := range r.metadata.Preloads {
		orm = orm.Preload(preload)
	}

	err = orm.
		Create(values).
		Error

	if db.IsDuplicateEntryError(err) {
		logger.Warn("could not create models of type %s due to duplicate entry error: %s", modelId, err.Error())
		return &db.DuplicateEntryError{
			Err: err,
		}
	}

	if err != nil {
		logger.Error("could not create model of type %v: %w", modelId, err)
		return err
	}

	for _, value := range valuesSlice {
		vm := value.(ModelBased)

		if err := r.refreshAssociations(ctx, vm.(ModelBased), Create); err != nil {
			logger.Error("could not refresh associations of type %s with id %d: %w", modelId, mdl.EmptyIfNil(vm.GetId()), err)
			return err
		}
	}

	for _, v := range valuesSlice {
		logger.Info("created model of type %s with id %d", modelId, *v.(ModelBased).GetId())
	}

	return nil
}

func (r *repository) BatchUpdate(ctx context.Context, values interface{}) error {
	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)
	if err != nil {
		return fmt.Errorf("could not turn values into slice: %w", err)
	}

	if len(valuesSlice) == 0 {
		return nil
	}

	queryable, err := r.isQueryableModel(valuesSlice[0])
	if err != nil {
		return err
	}

	if !queryable {
		return ErrCrossUpdate
	}

	valueType := reflect.TypeOf(valuesSlice[0])
	for i := 0; i < len(valuesSlice); i++ {
		if _, ok := valuesSlice[i].(ModelBased); !ok {
			return fmt.Errorf("you should pass a slice of ModelBased, found element at %d with %T", i, valuesSlice[0])
		}

		if valueType != reflect.TypeOf(valuesSlice[i]) {
			return fmt.Errorf("your elements should have all the same types, %d was different", i)
		}
	}

	modelId := r.GetModelId()
	logger := r.logger.WithContext(ctx)

	ctx, span := r.startSubSpan(ctx, "UpdateItems")
	defer span.Finish()

	orm := r.orm.
		WithContext(ctx).
		Session(&gorm.Session{
			FullSaveAssociations: true,
		})

	for _, preload := range r.metadata.Preloads {
		orm = orm.Preload(preload)
	}

	err = orm.
		Save(values).
		Error

	if db.IsDuplicateEntryError(err) {
		logger.Warn("could not update models of type %s due to duplicate entry error: %s", modelId, err.Error())
		return &db.DuplicateEntryError{
			Err: err,
		}
	}

	if err != nil {
		logger.Error("could not update models of type %v: %w", modelId, err)
		return err
	}

	for _, value := range valuesSlice {
		vm := value.(ModelBased)

		if err := r.refreshAssociations(ctx, vm.(ModelBased), Update); err != nil {
			logger.Error("could not refresh associations of type %s with id %d: %w", modelId, mdl.EmptyIfNil(vm.GetId()), err)
			return err
		}
	}

	for _, v := range valuesSlice {
		logger.Info("updated model of type %s with id %d", modelId, *v.(ModelBased).GetId())
	}

	return nil
}

func (r *repository) BatchDelete(ctx context.Context, values interface{}) error {
	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)
	if err != nil {
		return fmt.Errorf("could not turn values into slice: %w", err)
	}

	if len(valuesSlice) == 0 {
		return nil
	}

	valueType := reflect.TypeOf(valuesSlice[0])

	for i := 0; i < len(valuesSlice); i++ {
		if _, ok := valuesSlice[i].(ModelBased); !ok {
			return fmt.Errorf("you should pass a slice of ModelBased, found element at %d with %T", i, valuesSlice[0])
		}

		if valueType != reflect.TypeOf(valuesSlice[i]) {
			return fmt.Errorf("your elements have all the same types, %d was different", i)
		}
	}

	queryable, err := r.isQueryableModel(valuesSlice[0])
	if err != nil {
		return err
	}

	if !queryable {
		return ErrCrossDelete
	}

	modelId := r.GetModelId()
	logger := r.logger.WithContext(ctx)

	ctx, span := r.startSubSpan(ctx, "DeleteItems")
	defer span.Finish()

	for _, value := range valuesSlice {
		vm := value.(ModelBased)

		if err := r.refreshAssociations(ctx, vm.(ModelBased), Delete); err != nil {
			logger.Error("could not refresh associations of type %s with id %d: %w", modelId, mdl.EmptyIfNil(vm.GetId()), err)
			return err
		}
	}

	orm := r.orm.
		WithContext(ctx).
		Session(&gorm.Session{
			FullSaveAssociations: true,
		})

	for _, preload := range r.metadata.Preloads {
		orm = orm.Preload(preload)
	}

	err = orm.
		Delete(values).
		Error

	if err != nil {
		logger.Error("could not delete models of type %v: %w", modelId, err)
		return err
	}

	for _, v := range valuesSlice {
		logger.Info("deleted model of type %s with id %d", modelId, *v.(ModelBased).GetId())
	}

	return nil
}

func (r *repository) Create(ctx context.Context, value ModelBased) error {
	queryable, err := r.isQueryableModel(value)
	if err != nil {
		return err
	}

	if !queryable {
		return ErrCrossCreate
	}

	modelId := r.GetModelId()
	logger := r.logger.WithContext(ctx)

	ctx, span := r.startSubSpan(ctx, "Create")
	defer span.Finish()

	orm := r.orm.
		WithContext(ctx).
		Session(&gorm.Session{
			FullSaveAssociations: true,
		})

	for _, preload := range r.metadata.Preloads {
		orm = orm.Preload(preload)
	}

	err = orm.
		Create(value).
		Error

	if db.IsDuplicateEntryError(err) {
		logger.Warn("could not create model of type %s due to duplicate entry error: %s", modelId, err.Error())
		return &db.DuplicateEntryError{
			Err: err,
		}
	}

	if err != nil {
		logger.Error("could not create model of type %s: %w", modelId, err)
		return err
	}

	err = r.refreshAssociations(ctx, value, Create)
	if err != nil {
		logger.Error("could not refresh associations of model type %s: %w", modelId, err)
	}

	logger.Info("created model of type %s with id %d", modelId, *value.GetId())

	return nil
}

func (r *repository) Read(ctx context.Context, id *uint, out ModelBased) error {
	queryable, err := r.isQueryableModel(out)
	if err != nil {
		return err
	}

	if !queryable {
		return ErrCrossRead
	}

	modelId := r.GetModelId()

	_, span := r.startSubSpan(ctx, "Get")
	defer span.Finish()

	orm := r.orm.
		WithContext(ctx).
		Session(&gorm.Session{
			FullSaveAssociations: true,
		})

	for _, preload := range r.metadata.Preloads {
		orm = orm.Preload(preload)
	}

	err = orm.
		First(out, *id).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return NewRecordNotFoundError(*id, modelId, err)
	}

	return err
}

func (r *repository) Update(ctx context.Context, value ModelBased) error {
	queryable, err := r.isQueryableModel(value)
	if err != nil {
		return err
	}

	if !queryable {
		return ErrCrossUpdate
	}

	modelId := r.GetModelId()
	logger := r.logger.WithContext(ctx)

	ctx, span := r.startSubSpan(ctx, "UpdateItem")
	defer span.Finish()

	orm := r.orm.
		WithContext(ctx).
		Session(&gorm.Session{
			FullSaveAssociations: true,
		})

	for _, preload := range r.metadata.Preloads {
		orm = orm.Preload(preload)
	}

	err = orm.
		Save(value).
		Error

	if db.IsDuplicateEntryError(err) {
		logger.Warn("could not update model of type %s with id %d due to duplicate entry error: %s", modelId, mdl.EmptyIfNil(value.GetId()), err.Error())
		return &db.DuplicateEntryError{
			Err: err,
		}
	}

	if err != nil {
		logger.Error("could not update model of type %s with id %d: %w", modelId, mdl.EmptyIfNil(value.GetId()), err)
		return err
	}

	if err := r.refreshAssociations(ctx, value, Update); err != nil {
		logger.Error("could not refresh associations of type %s with id %d: %w", modelId, mdl.EmptyIfNil(value.GetId()), err)
		return err
	}

	logger.Info("updated model of type %s with id %d", modelId, *value.GetId())

	return nil
}

func (r *repository) Delete(ctx context.Context, value ModelBased) error {
	queryable, err := r.isQueryableModel(value)
	if err != nil {
		return err
	}

	if !queryable {
		return ErrCrossDelete
	}

	modelId := r.GetModelId()
	logger := r.logger.WithContext(ctx)

	_, span := r.startSubSpan(ctx, "Delete")
	defer span.Finish()

	if err := r.refreshAssociations(ctx, value, Delete); err != nil {
		logger.Error("could not refresh associations of type %s with id %d: %w", modelId, mdl.EmptyIfNil(value.GetId()), err)
		return err
	}

	err = r.orm.
		WithContext(ctx).
		Session(&gorm.Session{
			FullSaveAssociations: true,
		}).
		Delete(value).
		Error

	if err != nil {
		logger.Error("could not delete model of type %s with id %d: %w", modelId, *value.GetId(), err)
	}

	logger.Info("deleted model of type %s with id %d", modelId, *value.GetId())

	return err
}

func (r *repository) Query(ctx context.Context, qb *QueryBuilder, result interface{}) error {
	err := r.checkResultModel(result)
	if err != nil {
		return ErrCrossQuery
	}

	_, span := r.startSubSpan(ctx, "Query")
	defer span.Finish()

	db := r.orm.WithContext(ctx)

	for _, j := range qb.joins {
		db = db.Joins(j)
	}

	for i := range qb.where {
		currentWhere := qb.where[i]
		if reflect.TypeOf(currentWhere).Kind() == reflect.Ptr ||
			reflect.TypeOf(currentWhere).Kind() == reflect.Struct {

			queryable, err := r.isQueryableModel(currentWhere)
			if err != nil {
				return err
			}

			if !queryable {
				return ErrCrossQuery
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

	for _, p := range r.metadata.Preloads {
		db = db.Preload(p)
	}

	if qb.page != nil {
		db = db.Offset(qb.page.offset)
		db = db.Limit(qb.page.limit)
	}

	db = db.Table(r.GetMetadata().TableName)

	err = db.Find(result).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return NewNoQueryResultsError(r.GetModelId(), err)
	}

	return err
}

func (r *repository) Count(ctx context.Context, qb *QueryBuilder, model ModelBased) (int, error) {
	_, span := r.startSubSpan(ctx, "Count")
	defer span.Finish()

	db := r.orm.WithContext(ctx)

	for _, j := range qb.joins {
		db = db.Joins(j)
	}

	for i := range qb.where {
		db = db.Where(qb.where[i], qb.args[i]...)
	}

	var count int64
	tx := db.Model(model).Count(&count)

	return int(count), tx.Error
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

func ParsePreloads(model interface{}) []string {
	return parsePreloadsForModel("", model)
}

func parsePreloadsForModel(parent string, model interface{}) []string {
	value := reflect.ValueOf(model)
	if value.Kind() == reflect.Ptr && value.IsNil() {
		value = reflect.New(value.Type().Elem())
	}

	modelType := reflect.Indirect(value).Type()
	if modelType.Kind() == reflect.Interface {
		modelType = reflect.Indirect(reflect.ValueOf(model)).Elem().Type()
	}

	for modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array || modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	preloads := make([]string, 0)

	for i := 0; i < modelType.NumField(); i++ {
		fieldStruct := modelType.Field(i)
		if !ast.IsExported(fieldStruct.Name) {
			continue
		}

		tags := fieldStruct.Tag.Get("orm")
		parts := strings.Split(tags, ";")

		if slices.Contains(parts, "preload:false") {
			continue
		}

		if !slices.Contains(parts, "preload") {
			continue
		}

		fieldKind := fieldStruct.Type.Kind()
		if fieldKind != reflect.Ptr &&
			fieldKind != reflect.Struct &&
			fieldKind != reflect.Array &&
			fieldKind != reflect.Slice {
			continue
		}

		currentName := fieldStruct.Name
		if parent != "" {
			currentName = parent + "." + currentName
		}

		if strings.HasSuffix(parent, fieldStruct.Name) {
			// don't allow self-references over one level as it ends in infinite recursion
			continue
		}

		preloads = append(preloads, currentName)

		fieldType := fieldStruct.Type
		for fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		fieldValue := reflect.New(fieldType).Interface()

		childNames := parsePreloadsForModel(currentName, fieldValue)

		for _, child := range childNames {
			preloads = append(preloads, child)
		}
	}

	return preloads
}

func (r *repository) checkResultModel(result interface{}) error {
	if !refl.IsPointerToSlice(result) {
		return fmt.Errorf("result slice has to be pointer to slice")
	}

	model := reflect.ValueOf(result).Elem().Interface()

	queryable, err := r.isQueryableModel(model)
	if err != nil {
		return err
	}

	if !queryable {
		return fmt.Errorf("cross querying result slice has to be of same model")
	}

	return nil
}

func (r *repository) isQueryableModel(model interface{}) (bool, error) {
	scheme, err := schema.Parse(model, r.schemaCache, r.orm.NamingStrategy)
	if err != nil {
		return false, fmt.Errorf("could not parse model: %w", err)
	}

	return strings.EqualFold(scheme.Table, r.GetMetadata().TableName) || scheme.Table == "", nil
}

func (r *repository) startSubSpan(ctx context.Context, action string) (context.Context, tracing.Span) {
	modelName := r.GetModelId()
	spanName := fmt.Sprintf("db_repo.%v.%v", modelName, action)

	ctx, span := r.tracer.StartSubSpan(ctx, spanName)
	span.AddMetadata("model", modelName)

	return ctx, span
}

func (r *repository) refreshAssociations(ctx context.Context, value ModelBased, op string) error {
	scheme, err := schema.Parse(value, r.schemaCache, r.orm.NamingStrategy)
	if err != nil {
		return fmt.Errorf("could not parse schema: %w", err)
	}

	of := reflect.ValueOf(value)
	if of.Kind() != reflect.Ptr {
		return fmt.Errorf("you must pass a pointer to your repository method")
	}

	e := of.Elem()

	orm := r.orm.WithContext(ctx)

	for _, rel := range scheme.Relationships.Relations {
		v := e.FieldByName(rel.Name)

		tags := rel.Field.Tag.Get("orm")
		parts := strings.Split(tags, ";")

		if !slices.Contains(parts, "assoc_update") {
			continue
		}

		switch op {
		case Create:
			fallthrough

		case Update:
			switch rel.Type {
			case schema.Many2Many:
				err = orm.Model(value).Association(rel.Name).Replace(v.Interface())
			default:
				assocIds := readIdsFromReflectValue(v)
				parentId := value.GetId()

				tableName := rel.FieldSchema.Table
				args := make([]interface{}, 0)

				qry := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", tableName, rel.References[0].ForeignKey.DBName)
				args = append(args, parentId)

				if len(assocIds) != 0 {
					qry = qry + fmt.Sprintf(" AND %s NOT IN ?", "id")
					args = append(args, assocIds)
				}

				err = orm.Exec(qry, args...).Error
			}
		case Delete:
			switch rel.Type {
			case schema.HasMany:
				id := e.FieldByName("Id").Elem().Interface()

				tableName := rel.FieldSchema.Table
				qry := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", tableName, rel.References[0].ForeignKey.DBName)
				err = r.orm.Exec(qry, id).Error
			default:
				err = orm.Model(value).Association(rel.Name).Clear()
			}
		default:
			err = fmt.Errorf("unknown operation %s", op)
		}
	}

	if err != nil {
		return err
	}

	if op == Delete || len(scheme.Relationships.Relations) == 0 {
		return nil
	}

	return r.Read(ctx, value.GetId(), value)
}

func readIdsFromReflectValue(values reflect.Value) []*uint {
	ids := make([]*uint, 0)

	for j := 0; j < values.Len(); j++ {
		value := values.Index(j)
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}

		id := value.FieldByName("Id").Interface().(*uint)
		ids = append(ids, id)
	}

	return ids
}
