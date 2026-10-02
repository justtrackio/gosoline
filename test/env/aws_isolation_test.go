//go:build integration

package env_test

import (
	"bytes"
	"context"
	"net"
	"net/url"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/require"
)

func TestSharedAwsServicesPurgeIsolation(t *testing.T) {
	settings := func() map[string]any {
		return map[string]any{
			"app.env": "test", "app.name": "aws-isolation", "app.namespace": "{app.env}",
			"test.auto_detect.enabled":         false,
			"test.components.dynamodb.default": map[string]any{},
			"test.components.s3.default":       map[string]any{},
			"blob.default.bucket":              "shared-test-bucket",
		}
	}
	owner, err := env.NewEnvironment(t, env.WithConfigMap(settings()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, owner.Stop()) })
	externalSettings := settings()
	externalSettings["test.container_manager.runner_type"] = "external"
	for typ, address := range map[string]string{"dynamodb": owner.DynamoDb("default").Address(), "s3": owner.S3("default").Address()} {
		endpoint, err := url.Parse(address)
		require.NoError(t, err)
		host, port, err := net.SplitHostPort(endpoint.Host)
		require.NoError(t, err)
		portNumber, err := strconv.Atoi(port)
		require.NoError(t, err)
		externalSettings["test.components."+typ+".default.host"] = host
		externalSettings["test.components."+typ+".default.port"] = portNumber
	}
	external, err := env.NewEnvironment(t, env.WithConfigMap(externalSettings))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, external.Stop()) })
	ctx := context.Background()
	tables := make([]string, 0, 2)
	buckets := make([]string, 0, 2)
	for _, environment := range []*env.Environment{owner, external} {
		table, err := ddb.GetTableName(environment.Config(), &ddb.Settings{ModelId: mdl.ModelId{Env: "test", Name: "items"}})
		require.NoError(t, err)
		tables = append(tables, table)
		client := environment.DynamoDb("default").DdbClient()
		_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
			TableName:             aws.String(table),
			AttributeDefinitions:  []ddbTypes.AttributeDefinition{{AttributeName: aws.String("id"), AttributeType: ddbTypes.ScalarAttributeTypeS}},
			KeySchema:             []ddbTypes.KeySchemaElement{{AttributeName: aws.String("id"), KeyType: ddbTypes.KeyTypeHash}},
			ProvisionedThroughput: &ddbTypes.ProvisionedThroughput{ReadCapacityUnits: aws.Int64(1), WriteCapacityUnits: aws.Int64(1)},
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(table)})
			require.NoError(t, err)
		})
		_, err = client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(table), Item: map[string]ddbTypes.AttributeValue{"id": &ddbTypes.AttributeValueMemberS{Value: "same-key"}}})
		require.NoError(t, err)

		storeSettings, err := blob.ReadStoreSettings(environment.Config(), "default")
		require.NoError(t, err)
		buckets = append(buckets, storeSettings.Bucket)
		service, err := blob.NewService(environment.Context(), environment.Config(), environment.Logger(), storeSettings)
		require.NoError(t, err)
		require.NoError(t, service.CreateBucket(ctx))
		s3Client := environment.S3("default").S3Client()
		t.Cleanup(func() {
			require.NoError(t, service.Purge(ctx))
			_, err := s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(storeSettings.Bucket)})
			require.NoError(t, err)
		})
		_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(storeSettings.Bucket), Key: aws.String("same-key"), Body: bytes.NewReader([]byte("value"))})
		require.NoError(t, err)
	}
	require.NotEqual(t, tables[0], tables[1])
	require.NotEqual(t, buckets[0], buckets[1])
	purger, err := ddb.NewLifeCyclePurger(owner.Context(), owner.Config(), owner.Logger(), "default", tables[0])
	require.NoError(t, err)
	require.NoError(t, purger.Purge(ctx))
	item, err := external.DynamoDb("default").DdbClient().GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(tables[1]), Key: map[string]ddbTypes.AttributeValue{"id": &ddbTypes.AttributeValueMemberS{Value: "same-key"}}})
	require.NoError(t, err)
	require.NotEmpty(t, item.Item)
	ownerStore, err := blob.ReadStoreSettings(owner.Config(), "default")
	require.NoError(t, err)
	ownerService, err := blob.NewService(owner.Context(), owner.Config(), owner.Logger(), ownerStore)
	require.NoError(t, err)
	require.NoError(t, ownerService.Purge(ctx))
	objects, err := external.S3("default").S3Client().ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(buckets[1])})
	require.NoError(t, err)
	require.Len(t, objects.Contents, 1)
}
