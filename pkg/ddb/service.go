package ddb

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoDynamodb "github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type TableDescription struct {
	Name      string
	ItemCount int64
}

type Service struct {
	logger          log.Logger
	client          gosoDynamodb.Client
	metadataFactory *MetadataFactory
}

func NewService(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings, optFns ...gosoDynamodb.ClientOption) (*Service, error) {
	sanitizeSettings(settings)

	var err error
	var client gosoDynamodb.Client

	if client, err = gosoDynamodb.ProvideClient(ctx, config, logger, settings.ClientName, optFns...); err != nil {
		return nil, fmt.Errorf("can not create dynamodb client: %w", err)
	}

	metadataFactory := NewMetadataFactory(config, settings)

	return NewServiceWithInterfaces(logger, client, metadataFactory), nil
}

func NewServiceWithInterfaces(logger log.Logger, client gosoDynamodb.Client, metadataFactory *MetadataFactory) *Service {
	return &Service{
		logger:          logger,
		client:          client,
		metadataFactory: metadataFactory,
	}
}

func (s *Service) DescribeTable(ctx context.Context) (*TableDescription, error) {
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(s.metadataFactory.GetTableName()),
	}

	out, err := s.client.DescribeTable(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("can not get table description: %w", err)
	}

	description := &TableDescription{
		Name:      s.metadataFactory.GetTableName(),
		ItemCount: mdl.EmptyIfNil(out.Table.ItemCount),
	}

	return description, nil
}

func (s *Service) CreateTable(ctx context.Context) (*Metadata, error) {
	var err error
	var metadata *Metadata
	var exists bool

	if metadata, err = s.metadataFactory.GetMetadata(); err != nil {
		return nil, fmt.Errorf("can not get metadata: %w", err)
	}

	if exists, err = s.tableExists(ctx, s.metadataFactory.GetTableName()); err != nil {
		return nil, fmt.Errorf("can not check if the table already exists: %w", err)
	}

	if exists {
		return metadata, nil
	}

	settings := s.metadataFactory.GetSettings()

	if !exists && !settings.AutoCreate {
		return nil, fmt.Errorf("table does not exist and auto create is disabled")
	}

	mainKeySchema, err := s.getKeySchema(metadata.Main)
	if err != nil {
		return metadata, fmt.Errorf("can not create main key schema for table %s: %w", s.metadataFactory.GetTableName(), err)
	}

	localIndices, err := s.getLocalSecondaryIndices(metadata)
	if err != nil {
		return metadata, fmt.Errorf("can not create definitions for local secondary indices on table %s: %w", s.metadataFactory.GetTableName(), err)
	}

	globalIndices, err := s.getGlobalSecondaryIndices(metadata)
	if err != nil {
		return metadata, fmt.Errorf("can not create definitions for global secondary indices on table %s: %w", s.metadataFactory.GetTableName(), err)
	}

	attributeDefinitions := s.getAttributeDefinitions(metadata)

	streamSpecification := &types.StreamSpecification{
		StreamEnabled: aws.Bool(false),
	}

	if settings.Main.StreamView != "" {
		streamSpecification.StreamEnabled = aws.Bool(true)
		streamSpecification.StreamViewType = settings.Main.StreamView
	}

	input := &dynamodb.CreateTableInput{
		TableName:              aws.String(s.metadataFactory.GetTableName()),
		AttributeDefinitions:   attributeDefinitions,
		KeySchema:              mainKeySchema,
		LocalSecondaryIndexes:  localIndices,
		GlobalSecondaryIndexes: globalIndices,
		StreamSpecification:    streamSpecification,
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(metadata.Main.ReadCapacityUnits),
			WriteCapacityUnits: aws.Int64(metadata.Main.WriteCapacityUnits),
		},
	}

	_, err = s.client.CreateTable(ctx, input)

	var errResourceInUseException *types.ResourceInUseException
	if errors.As(err, &errResourceInUseException) {
		return metadata, nil
	}

	if err != nil {
		return nil, fmt.Errorf("could not create table: %w", err)
	}

	if err = s.waitForTableGettingAvailable(ctx, s.metadataFactory.GetTableName()); err != nil {
		return nil, err
	}

	s.logger.Info("created ddb table %s", s.metadataFactory.GetTableName())

	err = s.updateTtlSpecification(ctx, metadata)

	return metadata, err
}

func (s *Service) updateTtlSpecification(ctx context.Context, metadata *Metadata) error {
	ttlSpecification, err := s.getTimeToLiveSpecification(metadata)
	if err != nil {
		return fmt.Errorf("can not create ttl specification for table %s: %w", metadata.TableName, err)
	}

	if ttlSpecification == nil {
		return nil
	}

	ttlInput := &dynamodb.UpdateTimeToLiveInput{
		TableName:               aws.String(metadata.TableName),
		TimeToLiveSpecification: ttlSpecification,
	}

	for i := 0; i < defaultMaxWaitSeconds; i++ {
		_, err = s.client.UpdateTimeToLive(ctx, ttlInput)

		var errResourceInUseException *types.ResourceInUseException
		if errors.As(err, &errResourceInUseException) {
			time.Sleep(time.Second)

			continue
		}

		if err != nil {
			return fmt.Errorf("could not update ttl specification for ddb table %s: %w", metadata.TableName, err)
		}

		s.logger.Info("updated ttl specification for ddb table %s", metadata.TableName)

		return nil
	}

	return fmt.Errorf("could not update ttl specification for ddb table %s cause the table is still in use", metadata.TableName)
}

func (s *Service) getAttributeDefinitions(metadata *Metadata) []types.AttributeDefinition {
	definitions := make([]types.AttributeDefinition, 0)
	keyFields := s.getKeyFields(metadata)

	for _, field := range keyFields {
		attr := metadata.Attributes[field]

		definitions = append(definitions, types.AttributeDefinition{
			AttributeName: aws.String(attr.AttributeName),
			AttributeType: attr.Type,
		})
	}

	return definitions
}

func (s *Service) getKeyFields(metadata *Metadata) []string {
	fields := make([]string, 0)
	fields = append(fields, metadata.Main.GetKeyFields()...)

	for _, data := range metadata.Local {
		fields = append(fields, data.GetKeyFields()...)
	}

	for _, data := range metadata.Global {
		fields = append(fields, data.GetKeyFields()...)
	}

	fields = funk.Uniq(fields)
	sort.Strings(fields)

	return fields
}

func (s *Service) getKeySchema(metadata KeyAware) ([]types.KeySchemaElement, error) {
	schema := make([]types.KeySchemaElement, 0)

	if metadata.GetHashKey() == nil {
		return schema, fmt.Errorf("no hash key defined")
	}

	schema = append(schema, types.KeySchemaElement{
		AttributeName: metadata.GetHashKey(),
		KeyType:       types.KeyTypeHash,
	})

	if metadata.GetRangeKey() == nil {
		return schema, nil
	}

	schema = append(schema, types.KeySchemaElement{
		AttributeName: metadata.GetRangeKey(),
		KeyType:       types.KeyTypeRange,
	})

	return schema, nil
}

func (s *Service) getLocalSecondaryIndices(meta *Metadata) ([]types.LocalSecondaryIndex, error) {
	if len(meta.Local) == 0 {
		return nil, nil
	}

	names := make([]string, 0)
	indices := make([]types.LocalSecondaryIndex, 0, len(meta.Local))

	for name := range meta.Local {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		data := meta.Local[name]
		keySchema, err := s.getKeySchema(data)
		if err != nil {
			return nil, err
		}

		projection, err := s.projectedFields(meta.Main, data)
		if err != nil {
			return nil, err
		}

		indices = append(indices, types.LocalSecondaryIndex{
			IndexName:  aws.String(name),
			KeySchema:  keySchema,
			Projection: projection,
		})
	}

	return indices, nil
}

func (s *Service) getGlobalSecondaryIndices(meta *Metadata) ([]types.GlobalSecondaryIndex, error) {
	if len(meta.Global) == 0 {
		return nil, nil
	}

	names := make([]string, 0)
	indices := make([]types.GlobalSecondaryIndex, 0, len(meta.Local))

	for name := range meta.Global {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		data := meta.Global[name]
		keySchema, err := s.getKeySchema(data)
		if err != nil {
			return nil, err
		}

		projection, err := s.projectedFields(meta.Main, data)
		if err != nil {
			return nil, err
		}

		indices = append(indices, types.GlobalSecondaryIndex{
			IndexName:  aws.String(name),
			KeySchema:  keySchema,
			Projection: projection,
			ProvisionedThroughput: &types.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(data.ReadCapacityUnits),
				WriteCapacityUnits: aws.Int64(data.WriteCapacityUnits),
			},
		})
	}

	return indices, nil
}

func (s *Service) projectedFields(main FieldAware, second FieldAware) (*types.Projection, error) {
	mainFields := main.GetFields()
	secondFields := second.GetFields()

	for _, field := range secondFields {
		if !slices.Contains(mainFields, field) {
			return nil, fmt.Errorf("can't project field '%s' cause the main table is missing this field", field)
		}
	}

	if len(mainFields) == len(secondFields) {
		projection := &types.Projection{
			NonKeyAttributes: nil,
			ProjectionType:   types.ProjectionTypeAll,
		}

		return projection, nil
	}

	projected := make([]string, 0)

	for _, field := range secondFields {
		if main.IsKeyField(field) || second.IsKeyField(field) {
			continue
		}

		projected = append(projected, field)
	}

	if len(projected) == 0 {
		projection := &types.Projection{
			NonKeyAttributes: nil,
			ProjectionType:   types.ProjectionTypeKeysOnly,
		}

		return projection, nil
	}

	projection := &types.Projection{
		NonKeyAttributes: projected,
		ProjectionType:   types.ProjectionTypeInclude,
	}

	return projection, nil
}

func (s *Service) getTimeToLiveSpecification(metadata *Metadata) (*types.TimeToLiveSpecification, error) {
	if !metadata.TimeToLive.Enabled {
		return nil, nil
	}

	attr := metadata.Attributes[metadata.TimeToLive.Field]

	if attr.Type != types.ScalarAttributeTypeN {
		return nil, fmt.Errorf("the attribute of the ttl field '%s' has to be of type N but instead is of type %s ", attr.FieldName, attr.Type)
	}

	specification := &types.TimeToLiveSpecification{
		Enabled:       aws.Bool(true),
		AttributeName: aws.String(attr.AttributeName),
	}

	return specification, nil
}

func (s *Service) waitForTableGettingAvailable(ctx context.Context, name string) error {
	s.logger.Info("waiting for ddb table %s getting available", name)

	for i := 0; i < defaultMaxWaitSeconds; i++ {
		exists, err := s.checkStatus(ctx, name, tableAvailableMapping)
		if err != nil {
			return err
		}

		if exists {
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("table %s was not getting available in time", name)
}

func (s *Service) tableExists(ctx context.Context, name string) (bool, error) {
	s.logger.Info("looking for ddb table %v", name)

	exists, err := s.checkStatus(ctx, name, tableExistingMapping)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	s.logger.Info("found ddb table %s", name)

	return true, nil
}

type tableStatusMapping struct {
	result bool
	err    error
}

func statusError(msg string) tableStatusMapping {
	return tableStatusMapping{
		err: fmt.Errorf("%s", msg),
	}
}

func statusResult(result bool) tableStatusMapping {
	return tableStatusMapping{
		result: result,
	}
}

var (
	tableAvailableMapping = map[types.TableStatus]tableStatusMapping{
		types.TableStatusCreating:                          statusResult(false),
		types.TableStatusUpdating:                          statusResult(true),
		types.TableStatusActive:                            statusResult(true),
		types.TableStatusDeleting:                          statusError("can not access deleting table"),
		types.TableStatusInaccessibleEncryptionCredentials: statusError("table is not accessible because of inaccessible encryption details"),
		types.TableStatusArchived:                          statusError("can not access archived table"),
		types.TableStatusArchiving:                         statusError("can not access archiving table"),
	}
	tableExistingMapping = map[types.TableStatus]tableStatusMapping{
		types.TableStatusCreating:                          statusResult(true),
		types.TableStatusUpdating:                          statusResult(true),
		types.TableStatusActive:                            statusResult(true),
		types.TableStatusDeleting:                          statusError("can not access deleting table"),
		types.TableStatusInaccessibleEncryptionCredentials: statusError("table is not accessible because of inaccessible encryption details"),
		types.TableStatusArchived:                          statusError("can not access archived table"),
		types.TableStatusArchiving:                         statusError("can not access archiving table"),
	}
)

func (s *Service) checkStatus(ctx context.Context, name string, statusMap map[types.TableStatus]tableStatusMapping) (bool, error) {
	out, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(name),
	})

	var errResourceNotFoundException *types.ResourceNotFoundException
	if errors.As(err, &errResourceNotFoundException) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("can not describe table: %w", err)
	}

	if result, ok := statusMap[out.Table.TableStatus]; ok {
		return result.result, result.err
	}

	return false, fmt.Errorf("unhandled table status %s", out.Table.TableStatus)
}
