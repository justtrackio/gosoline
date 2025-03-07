package ddb

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	gosoDynamodb "github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

type LifeCyclePurger struct {
	client         gosoDynamodb.Client
	clientSettings gosoDynamodb.ClientSettings
	tableName      string
}

func NewLifeCyclePurger(ctx context.Context, config cfg.Config, logger log.Logger, clientName string, tableName string) (*LifeCyclePurger, error) {
	var err error
	var client gosoDynamodb.Client

	if client, err = gosoDynamodb.ProvideClient(ctx, config, logger, clientName); err != nil {
		return nil, fmt.Errorf("can not create dynamodb client: %w", err)
	}

	clientSettings := gosoDynamodb.ClientSettings{}
	gosoAws.UnmarshalClientSettings(config, &clientSettings, "dynamodb", clientName)

	return &LifeCyclePurger{
		client:         client,
		clientSettings: clientSettings,
		tableName:      tableName,
	}, err
}

func (s *LifeCyclePurger) Purge(ctx context.Context) error {
	var err error
	var output *dynamodb.DescribeTableOutput

	describeInput := &dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName),
	}
	if output, err = s.client.DescribeTable(ctx, describeInput); err != nil {
		return fmt.Errorf("can not describe table: %w", err)
	}

	switch s.clientSettings.PurgeType {
	case "scan":
		return s.purgeScan(ctx, output.Table.KeySchema)
	case "drop_table":
		return s.purgeDropTable(ctx, output.Table)
	default:
		return fmt.Errorf("invalid purge type: %s", s.clientSettings.PurgeType)
	}
}

func (s *LifeCyclePurger) purgeDropTable(ctx context.Context, table *types.TableDescription) error {
	_, err := s.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(s.tableName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete table %s: %w", s.tableName, err)
	}

	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: table.AttributeDefinitions,
		KeySchema:            table.KeySchema,
		TableName:            table.TableName,
		GlobalSecondaryIndexes: funk.Map(table.GlobalSecondaryIndexes, func(index types.GlobalSecondaryIndexDescription) types.GlobalSecondaryIndex {
			return types.GlobalSecondaryIndex{
				IndexName:  index.IndexName,
				KeySchema:  index.KeySchema,
				Projection: index.Projection,
			}
		}),
		LocalSecondaryIndexes: funk.Map(table.LocalSecondaryIndexes, func(index types.LocalSecondaryIndexDescription) types.LocalSecondaryIndex {
			return types.LocalSecondaryIndex{
				IndexName:  index.IndexName,
				KeySchema:  index.KeySchema,
				Projection: index.Projection,
			}
		}),
	}

	if _, err = s.client.CreateTable(ctx, input); err != nil {
		return fmt.Errorf("failed to re-create table %s: %w", s.tableName, err)
	}

	return nil
}

func (s *LifeCyclePurger) purgeScan(ctx context.Context, keyFields []types.KeySchemaElement) error {
	cfn := coffin.New()

	totalSegments := s.clientSettings.PurgeParallelism
	if totalSegments == 0 {
		totalSegments = runtime.NumCPU()
	}

	cfn.GoWithContext(ctx, func(ctx context.Context) error {
		for i := range totalSegments {
			cfn.GoWithContext(ctx, func(ctx context.Context) error {
				return s.doPurgeScan(ctx, keyFields, i, totalSegments)
			})
		}

		return nil
	})

	if err := cfn.Wait(); err != nil {
		return fmt.Errorf("could not purge table %s: %w", s.tableName, err)
	}

	return nil
}

func (s *LifeCyclePurger) doPurgeScan(ctx context.Context, keyFields []types.KeySchemaElement, segment int, totalSegments int) error {
	var err error
	var out *dynamodb.ScanOutput

	tableName := aws.String(s.tableName)
	attributes := make([]string, len(keyFields))

	input := &dynamodb.ScanInput{
		Segment:                  aws.Int32(int32(segment)),
		TotalSegments:            aws.Int32(int32(totalSegments)),
		TableName:                tableName,
		ExpressionAttributeNames: map[string]string{},
	}

	for i, keyField := range keyFields {
		input.ExpressionAttributeNames[fmt.Sprintf("#%s", *keyField.AttributeName)] = *keyField.AttributeName
		attributes[i] = fmt.Sprintf("#%s", *keyField.AttributeName)
	}

	input.ProjectionExpression = aws.String(strings.Join(attributes, ","))

	for {
		if out, err = s.client.Scan(ctx, input); err != nil {
			return fmt.Errorf("can not get dynamodb scan: %w", err)
		}

		items := make([]types.WriteRequest, 0)

		for _, item := range out.Items {
			keys := make(map[string]types.AttributeValue)

			for _, keyField := range keyFields {
				keys[*keyField.AttributeName] = item[*keyField.AttributeName]
			}

			items = append(items, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{
					Key: keys,
				},
			})
		}

		chunks := funk.Chunk(items, 25)

		for _, chunk := range chunks {
			batchInput := &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]types.WriteRequest{
					*tableName: chunk,
				},
			}

			if _, err = s.client.BatchWriteItem(ctx, batchInput); err != nil {
				return fmt.Errorf("can not batch delete items: %w", err)
			}
		}

		if len(out.LastEvaluatedKey) == 0 {
			break
		}
	}

	return nil
}
