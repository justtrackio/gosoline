package ddb

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/thoas/go-funk"
	"sort"
	"time"
)

type TableDescription struct {
	Name      string
	ItemCount int64
}

type Service struct {
	logger          log.Logger
	client          dynamodbiface.DynamoDBAPI
	metadataFactory *metadataFactory
}

func NewService(config cfg.Config, logger log.Logger) *Service {
	client := cloud.GetDynamoDbClient(config, logger)

	return NewServiceWithInterfaces(logger, client)
}

func NewServiceWithInterfaces(logger log.Logger, client dynamodbiface.DynamoDBAPI) *Service {
	return &Service{
		logger:          logger,
		client:          client,
		metadataFactory: NewMetadataFactory(),
	}
}

func (s *Service) DescribeTable(settings *Settings) (*TableDescription, error) {
	tableName := TableName(settings)
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	out, err := s.client.DescribeTable(input)

	if err != nil {
		return nil, fmt.Errorf("can not get table description: %w", err)
	}

	description := &TableDescription{
		Name:      tableName,
		ItemCount: *out.Table.ItemCount,
	}

	return description, nil
}

func (s *Service) CreateTable(settings *Settings) (*Metadata, error) {
	tableName := TableName(settings)
	metadata, err := s.metadataFactory.GetMetadata(settings)

	if err != nil {
		return nil, err
	}

	if !settings.AutoCreate {
		return metadata, nil
	}

	exists, err := s.tableExists(tableName)

	if err != nil {
		return nil, err
	}

	if exists {
		return metadata, nil
	}

	mainKeySchema, err := s.getKeySchema(metadata.Main)

	if err != nil {
		return metadata, fmt.Errorf("can not create main key schema for table %s: %w", tableName, err)
	}

	localIndices, err := s.getLocalSecondaryIndices(metadata)

	if err != nil {
		return metadata, fmt.Errorf("can not create definitions for local secondary indices on table %s: %w", tableName, err)
	}

	globalIndices, err := s.getGlobalSecondaryIndices(metadata)

	if err != nil {
		return metadata, fmt.Errorf("can not create definitions for global secondary indices on table %s: %w", tableName, err)
	}

	attributeDefinitions := s.getAttributeDefinitions(metadata)

	streamSpecification := &dynamodb.StreamSpecification{
		StreamEnabled: aws.Bool(false),
	}

	if settings.Main.StreamView != "" {
		streamSpecification.StreamEnabled = aws.Bool(true)
		streamSpecification.StreamViewType = aws.String(settings.Main.StreamView)
	}

	input := &dynamodb.CreateTableInput{
		TableName:              aws.String(tableName),
		AttributeDefinitions:   attributeDefinitions,
		KeySchema:              mainKeySchema,
		LocalSecondaryIndexes:  localIndices,
		GlobalSecondaryIndexes: globalIndices,
		StreamSpecification:    streamSpecification,
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(metadata.Main.ReadCapacityUnits),
			WriteCapacityUnits: aws.Int64(metadata.Main.WriteCapacityUnits),
		},
	}

	_, err = s.client.CreateTable(input)

	if err != nil && isError(err, dynamodb.ErrCodeResourceInUseException) {
		return metadata, nil
	}

	if err != nil {
		return nil, err
	}

	err = s.waitForTableGettingAvailable(tableName)

	if err != nil {
		return nil, err
	}

	s.logger.Info("created ddb table %s", tableName)

	err = s.updateTtlSpecification(metadata)

	return metadata, err
}

func (s *Service) updateTtlSpecification(metadata *Metadata) error {
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
		_, err = s.client.UpdateTimeToLive(ttlInput)

		if isError(err, dynamodb.ErrCodeResourceInUseException) {
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

func (s *Service) getAttributeDefinitions(metadata *Metadata) []*dynamodb.AttributeDefinition {
	definitions := make([]*dynamodb.AttributeDefinition, 0)
	keyFields := s.getKeyFields(metadata)

	for _, field := range keyFields {
		attr := metadata.Attributes[field]

		definitions = append(definitions, &dynamodb.AttributeDefinition{
			AttributeName: aws.String(attr.AttributeName),
			AttributeType: aws.String(attr.Type),
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

	fields = funk.UniqString(fields)
	sort.Strings(fields)

	return fields
}

func (s *Service) getKeySchema(metadata KeyAware) ([]*dynamodb.KeySchemaElement, error) {
	schema := make([]*dynamodb.KeySchemaElement, 0)

	if metadata.GetHashKey() == nil {
		return schema, fmt.Errorf("no hash key defined")
	}

	schema = append(schema, &dynamodb.KeySchemaElement{
		AttributeName: metadata.GetHashKey(),
		KeyType:       aws.String(dynamodb.KeyTypeHash),
	})

	if metadata.GetRangeKey() == nil {
		return schema, nil
	}

	schema = append(schema, &dynamodb.KeySchemaElement{
		AttributeName: metadata.GetRangeKey(),
		KeyType:       aws.String(dynamodb.KeyTypeRange),
	})

	return schema, nil
}

func (s *Service) getLocalSecondaryIndices(meta *Metadata) ([]*dynamodb.LocalSecondaryIndex, error) {
	if len(meta.Local) == 0 {
		return nil, nil
	}

	names := make([]string, 0)
	indices := make([]*dynamodb.LocalSecondaryIndex, 0, len(meta.Local))

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

		indices = append(indices, &dynamodb.LocalSecondaryIndex{
			IndexName:  aws.String(name),
			KeySchema:  keySchema,
			Projection: projection,
		})
	}

	return indices, nil
}

func (s *Service) getGlobalSecondaryIndices(meta *Metadata) ([]*dynamodb.GlobalSecondaryIndex, error) {
	if len(meta.Global) == 0 {
		return nil, nil
	}

	names := make([]string, 0)
	indices := make([]*dynamodb.GlobalSecondaryIndex, 0, len(meta.Local))

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

		indices = append(indices, &dynamodb.GlobalSecondaryIndex{
			IndexName:  aws.String(name),
			KeySchema:  keySchema,
			Projection: projection,
			ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(data.ReadCapacityUnits),
				WriteCapacityUnits: aws.Int64(data.WriteCapacityUnits),
			},
		})
	}

	return indices, nil
}

func (s *Service) projectedFields(main FieldAware, second FieldAware) (*dynamodb.Projection, error) {
	mainFields := main.GetFields()
	secondFields := second.GetFields()

	for _, field := range secondFields {
		if !funk.Contains(mainFields, field) {
			return nil, fmt.Errorf("can't project field '%s' cause the main table is missing this field", field)
		}
	}

	if len(mainFields) == len(secondFields) {
		projection := &dynamodb.Projection{
			NonKeyAttributes: nil,
			ProjectionType:   aws.String(dynamodb.ProjectionTypeAll),
		}

		return projection, nil
	}

	projected := make([]*string, 0)

	for _, field := range secondFields {
		if main.IsKeyField(field) || second.IsKeyField(field) {
			continue
		}

		projected = append(projected, aws.String(field))
	}

	if len(projected) == 0 {
		projection := &dynamodb.Projection{
			NonKeyAttributes: nil,
			ProjectionType:   aws.String(dynamodb.ProjectionTypeKeysOnly),
		}

		return projection, nil
	}

	projection := &dynamodb.Projection{
		NonKeyAttributes: projected,
		ProjectionType:   aws.String(dynamodb.ProjectionTypeInclude),
	}

	return projection, nil
}

func (s *Service) getTimeToLiveSpecification(metadata *Metadata) (*dynamodb.TimeToLiveSpecification, error) {
	if !metadata.TimeToLive.Enabled {
		return nil, nil
	}

	attr := metadata.Attributes[metadata.TimeToLive.Field]

	if attr.Type != dynamodb.ScalarAttributeTypeN {
		return nil, fmt.Errorf("the attribute of the ttl field '%s' has to be of type N but instead is of type %s ", attr.FieldName, attr.Type)
	}

	specification := &dynamodb.TimeToLiveSpecification{
		Enabled:       aws.Bool(true),
		AttributeName: aws.String(attr.AttributeName),
	}

	return specification, nil
}

func (s *Service) waitForTableGettingAvailable(name string) error {
	s.logger.Info("waiting for ddb table %s getting available", name)

	for i := 0; i < defaultMaxWaitSeconds; i++ {
		exists, err := s.checkExists(name)

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

func (s *Service) tableExists(name string) (bool, error) {
	s.logger.Info("looking for ddb table %v", name)

	exists, err := s.checkExists(name)

	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	s.logger.Info("found ddb table %s", name)

	return true, nil
}

func (s *Service) checkExists(name string) (bool, error) {
	out, err := s.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(name),
	})

	if isError(err, dynamodb.ErrCodeResourceNotFoundException) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	active := *out.Table.TableStatus == dynamodb.TableStatusActive

	return active, nil
}
