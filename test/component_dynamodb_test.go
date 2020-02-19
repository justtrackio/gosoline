//+build integration

package test_test

import (
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_dynamodb(t *testing.T) {
	setup(t)

	pkgTest.Boot("test_configs/config.dynamodb.test.yml")
	defer pkgTest.Shutdown()

	ddbClient := pkgTest.ProvideDynamoDbClient("dynamodb")
	o, err := ddbClient.ListTables(&dynamodb.ListTablesInput{})

	assert.NoError(t, err)
	assert.Len(t, o.TableNames, 0)
}
