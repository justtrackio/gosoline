package ddb

import (
	"context"
	"errors"
	"fmt"
	"github.com/adjoeio/djoemo"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/guregu/dynamo"
	"reflect"
)

const (
	MetricNameAccessSuccess = "DdbAccessSuccess"
	MetricNameAccessFailure = "DdbAccessFailure"
	MetricNameAccessLatency = "DdbAccessLatency"

	OpSave = "save"
)

//go:generate mockery -name Repository
type Repository interface {
	GetModelId() mdl.ModelId
	CreateTable(model interface{}) error

	GetItem(ctx context.Context, qb QueryBuilder, result interface{}) (bool, error)
	GetItems(ctx context.Context, qb QueryBuilder, result interface{}) (bool, error)
	Query(ctx context.Context, qb QueryBuilder, result interface{}) error
	Save(ctx context.Context, item interface{}) error
	Update(ctx context.Context, exp djoemo.UpdateExpression, qb QueryBuilder, values map[string]interface{}) error
	QueryBuilder() QueryBuilder
}

//go:generate mockery -name DjoemoeRepository
type DjoemoeRepository interface {
	GIndex(name string) djoemo.GlobalIndexInterface
	GetItem(key djoemo.KeyInterface, item interface{}) (bool, error)
	GetItems(key djoemo.KeyInterface, items interface{}) (bool, error)
	GetItemWithContext(ctx context.Context, key djoemo.KeyInterface, out interface{}) (bool, error)
	GetItemsWithContext(ctx context.Context, key djoemo.KeyInterface, out interface{}) (bool, error)
	Query(query djoemo.QueryInterface, item interface{}) error
	QueryWithContext(ctx context.Context, query djoemo.QueryInterface, out interface{}) error
	SaveItemWithContext(ctx context.Context, key djoemo.KeyInterface, item interface{}) error
	UpdateWithContext(ctx context.Context, expression djoemo.UpdateExpression, key djoemo.KeyInterface, values map[string]interface{}) error
}

type QueryRepository interface {
	GetItem(key djoemo.KeyInterface, item interface{}) (bool, error)
	GetItems(key djoemo.KeyInterface, items interface{}) (bool, error)
	GetItemWithContext(ctx context.Context, key djoemo.KeyInterface, item interface{}) (bool, error)
	GetItemsWithContext(ctx context.Context, key djoemo.KeyInterface, items interface{}) (bool, error)
	Query(query djoemo.QueryInterface, item interface{}) error
	QueryWithContext(ctx context.Context, query djoemo.QueryInterface, item interface{}) error
}

var ErrNotFound = dynamo.ErrNotFound

type Settings struct {
	ModelId            mdl.ModelId
	AutoCreate         bool
	StreamView         string
	ReadCapacityUnits  int64
	WriteCapacityUnits int64
}

type repository struct {
	logger   mon.Logger
	tracer   tracing.Tracer
	client   dynamodbiface.DynamoDBAPI
	settings Settings

	db     *dynamo.DB
	table  *dynamo.Table
	exists bool

	djoemo DjoemoeRepository
}

func New(config cfg.Config, logger mon.Logger, settings Settings) *repository {
	settings.ModelId.PadFromConfig(config)
	settings.AutoCreate = config.GetBool("aws_dynamoDb_autoCreate")

	tracer := tracing.NewAwsTracer(config)
	client := cloud.GetDynamoDbClient(config, logger)
	dj := djoemo.NewRepository(client)

	return NewFromInterfaces(logger, tracer, client, settings, dj)
}

func NewFromInterfaces(logger mon.Logger, tracer tracing.Tracer, client dynamodbiface.DynamoDBAPI, settings Settings, dj DjoemoeRepository) *repository {
	name := getTableName(settings)

	db := dynamo.NewFromIface(client)
	t := db.Table(name)

	return &repository{
		logger:   logger,
		tracer:   tracer,
		client:   client,
		settings: settings,
		db:       db,
		table:    &t,

		djoemo: dj,
	}
}

func (r *repository) GetModelId() mdl.ModelId {
	return r.settings.ModelId
}

func (r *repository) CreateTable(model interface{}) error {
	if r.exists || r.settings.AutoCreate == false {
		return nil
	}

	name := getTableName(r.settings)

	if r.tableExists(name) {
		r.exists = true
		return nil
	}

	r.logger.Info(fmt.Sprintf("creating ddb table %v", name))

	ct := r.db.CreateTable(name, model)
	ct.Provision(r.settings.ReadCapacityUnits, r.settings.WriteCapacityUnits)

	if r.settings.StreamView != "" {
		ct.Stream(dynamo.StreamView(r.settings.StreamView))
	}

	err := ct.Run()

	if err != nil {
		r.logger.Error(err, fmt.Sprintf("could not create ddb table %v", name))
		return err
	}

	r.exists = true

	return nil
}

func (r *repository) GetItem(ctx context.Context, qb QueryBuilder, result interface{}) (bool, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.GetItem")
	defer span.Finish()

	qry := qb.Build()
	repo := r.getRepoByQuery(qb)
	found, err := repo.GetItemWithContext(ctx, qry, result)

	if err != nil {
		return false, err
	}

	return found, nil
}

func (r *repository) GetItems(ctx context.Context, qb QueryBuilder, result interface{}) (bool, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.GetItems")
	defer span.Finish()

	qry := qb.Build()
	repo := r.getRepoByQuery(qb)

	// GetItemsWithContext reports incorrectly that items were found even when the result is empty
	_, err := repo.GetItemsWithContext(ctx, qry, result)

	reflectValue := reflect.ValueOf(result)
	if reflectValue.Kind() != reflect.Ptr || reflectValue.Elem().Kind() != reflect.Slice {
		err := errors.New("result must be pointer to a slice")
		r.logger.WithContext(ctx).Error(err, err.Error())

		return false, err
	}

	found := reflectValue.Elem().Len() > 0

	return found, err
}

func (r *repository) Query(ctx context.Context, qb QueryBuilder, result interface{}) error {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.Query")
	defer span.Finish()

	qry := qb.Build()
	repo := r.getRepoByQuery(qb)
	err := repo.QueryWithContext(ctx, qry, result)

	return err
}

func (r *repository) Save(ctx context.Context, item interface{}) error {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.Save")
	defer span.Finish()

	qry := r.QueryBuilder().WithHash("", "").Build()
	err := r.djoemo.SaveItemWithContext(ctx, qry, item)

	return err
}

func (r *repository) Update(ctx context.Context, exp djoemo.UpdateExpression, qb QueryBuilder, values map[string]interface{}) error {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.Update")
	defer span.Finish()

	qry := qb.Build()

	return r.djoemo.UpdateWithContext(ctx, exp, qry, values)
}

func (r *repository) QueryBuilder() QueryBuilder {
	return NewQueryBuilder(getTableName(r.settings))
}

func (r *repository) getRepoByQuery(qb QueryBuilder) QueryRepository {
	index := qb.Index()

	if index == nil {
		return r.djoemo
	}

	return r.djoemo.GIndex(*index)
}

func (r *repository) tableExists(name string) bool {
	r.logger.Info(fmt.Sprintf("looking for ddb table %v", name))

	_, err := r.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(name),
	})

	return err == nil
}

func getTableName(s Settings) string {
	return fmt.Sprintf("%v-%v-%v-%v-%v", s.ModelId.Project, s.ModelId.Environment, s.ModelId.Family, s.ModelId.Application, s.ModelId.Name)
}
