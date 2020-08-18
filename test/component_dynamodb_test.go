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

	mocks, err := pkgTest.Boot("test_configs/config.dynamodb.test.yml")
	defer func() {
		if mocks != nil {
			mocks.Shutdown()
		}
	}()

	if err != nil {
		assert.Fail(t, "failed to boot mocks: %s", err.Error())

		return
	}

	ddbClient := mocks.ProvideDynamoDbClient("dynamodb")
	o, err := ddbClient.ListTables(&dynamodb.ListTablesInput{})

	assert.NoError(t, err)
	assert.Len(t, o.TableNames, 0)
}
