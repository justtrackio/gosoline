//+build integration

package dynamodb_test

import (
	"context"
	"errors"
	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/applike/gosoline/pkg/clock"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	gosoDdb "github.com/applike/gosoline/pkg/cloud/aws/dynamodb"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/test/suite"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsDdb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/mock"
	"testing"
)

type ClientTestSuite struct {
	suite.Suite
	clientDefault *awsDdb.Client
}

func (s *ClientTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithSharedEnvironment(),
		suite.WithConfigFile("client_test_cfg.yml"),
		suite.WithClockProvider(clock.NewRealClock()),
	}
}

func (s *ClientTestSuite) SetupTest() error {
	var err error
	var ctx = context.Background()
	var config = s.Env().Config()
	var logger = s.Env().Logger()

	credentials := awsCfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKID", "SECRET_KEY", "TOKEN"))
	if s.clientDefault, err = gosoDdb.NewClient(ctx, config, logger, "default", credentials); err != nil {
		return err
	}

	_, err = s.clientDefault.CreateTable(ctx, &awsDdb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
		TableName: aws.String("gosoline-cloud-dynamodb-test"),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
	})

	return err
}

func (s *ClientTestSuite) TearDownTest() error {
	_, err := s.clientDefault.DeleteTable(context.Background(), &awsDdb.DeleteTableInput{
		TableName: aws.String("gosoline-cloud-dynamodb-test"),
	})

	return err
}

func (s *ClientTestSuite) TestSuccess() {
	_, err := s.clientDefault.PutItem(context.Background(), &awsDdb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{
				Value: "goso-id",
			},
		},
		TableName: aws.String("gosoline-cloud-dynamodb-test"),
	})
	s.NoError(err)
}

func (s *ClientTestSuite) TestHttpTimeout() {
	proxy := s.Env().DynamoDb("default").Toxiproxy()
	proxy.AddToxic("latency_down", "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": 200,
	})

	ctx := context.Background()
	loggerMock := new(logMocks.Logger)
	loggerMock.On("WithContext", mock.Anything).Return(loggerMock)
	loggerMock.On("Warn", "attempt number %d to request resource %s after %s cause of error %s", mock.AnythingOfType("int"), "DynamoDB/PutItem", mock.AnythingOfType("time.Duration"), mock.AnythingOfType("*http.ResponseError")).Twice()
	loggerMock.On("Info", "sent request to resource %s successful after %d retries in %s", "DynamoDB/PutItem", 3, mock.AnythingOfType("time.Duration")).Once()

	credentials := awsCfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKID", "SECRET_KEY", "TOKEN"))
	client, err := gosoDdb.NewClient(ctx, s.Env().Config(), loggerMock, "http_timeout", credentials)
	s.NoError(err)

	_, err = client.PutItem(ctx, &awsDdb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{
				Value: "goso-id",
			},
		},
		TableName: aws.String("gosoline-cloud-dynamodb-test"),
	}, func(options *awsDdb.Options) {
		options.APIOptions = append(options.APIOptions, func(stack *middleware.Stack) error {
			i := 0

			return stack.Finalize.Add(middleware.FinalizeMiddlewareFunc("bla", func(ctx context.Context, input middleware.FinalizeInput, handler middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
				i++
				if i == 3 {
					proxy.RemoveToxic("latency_down")
				}

				return handler.HandleFinalize(ctx, input)
			}), middleware.After)
		})
	})

	s.NoError(err)
	loggerMock.AssertExpectations(s.T())
}

func (s *ClientTestSuite) TestMaxElapsedTimeExceeded() {
	proxy := s.Env().DynamoDb("default").Toxiproxy()
	proxy.AddToxic("latency_down", "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": 200,
	})

	ctx := context.Background()
	loggerMock := new(logMocks.Logger)
	loggerMock.On("WithContext", mock.Anything).Return(loggerMock)

	credentials := awsCfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKID", "SECRET_KEY", "TOKEN"))
	client, err := gosoDdb.NewClient(ctx, s.Env().Config(), loggerMock, "max_elapsed_time_exceeded", credentials)
	s.NoError(err)

	_, err = client.PutItem(ctx, &awsDdb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{
				Value: "goso-id",
			},
		},
		TableName: aws.String("gosoline-cloud-dynamodb-test"),
	})

	var expectedErr *gosoAws.ErrRetryMaxElapsedTimeExceeded
	isErrRetryAttemptsExceeded := errors.As(err, &expectedErr)

	s.True(isErrRetryAttemptsExceeded)
	loggerMock.AssertExpectations(s.T())
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
