//+build integration

package test_test

import (
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_dynamodb(t *testing.T) {
	setup(t)

	test.Boot(mdl.String("test_configs/config.dynamodb.test.yml"))

	ddbClient := test.ProvideDynamoDbClient("dynamodb")
	o, err := ddbClient.ListTables(&dynamodb.ListTablesInput{})

	assert.NoError(t, err)
	assert.Len(t, o.TableNames, 0)

	test.Shutdown()
}
