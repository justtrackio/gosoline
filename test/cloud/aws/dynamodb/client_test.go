//go:build integration

package dynamodb_test

import (
	"context"
	"testing"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsDdb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoDdb "github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb"
	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/stretchr/testify/mock"
)

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite
	clientDefault *awsDdb.Client
}

func (s *ClientTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithSharedEnvironment(),
		suite.WithConfigFile("client_test_cfg.yml"),
		suite.WithClockProvider(clock.NewRealClock()),
	}
}

func (s *ClientTestSuite) SetupTest() error {
	var err error
	ctx := context.Background()
	config := s.Env().Config()
	logger := s.Env().Logger()

	if s.clientDefault, err = gosoDdb.NewClient(ctx, config, logger, "default"); err != nil {
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
	_, err := proxy.AddToxic("latency_down", "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": 200,
	})
	s.NoError(err)

	ctx := context.Background()
	resource := &exec.ExecutableResource{
		Type: "DynamoDB",
		Name: "PutItem",
	}

	loggerMock := logMocks.NewLogger(s.T())
	loggerMock.EXPECT().WithContext(matcher.Context).Return(loggerMock)
	loggerMock.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(loggerMock)
	loggerMock.EXPECT().Warn("attempt number %d to request resource %s failed after %s cause of error: %s", mock.AnythingOfType("int"), resource, mock.AnythingOfType("time.Duration"), mock.AnythingOfType("*http.ResponseError")).Twice()
	loggerMock.EXPECT().Warn("sent request to resource %s successful after %d attempts in %s", resource, 3, mock.AnythingOfType("time.Duration")).Once()
	loggerMock.EXPECT().Info("created new %s client %s", "dynamodb", "http_timeout").Once()

	client, err := gosoDdb.NewClient(ctx, s.Env().Config(), loggerMock, "http_timeout")
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
					err := proxy.RemoveToxic("latency_down")
					s.NoError(err)
				}

				return handler.HandleFinalize(ctx, input)
			}), middleware.After)
		})
	})

	s.NoError(err)
}

func (s *ClientTestSuite) TestMaxElapsedTimeExceeded() {
	proxy := s.Env().DynamoDb("default").Toxiproxy()
	_, err := proxy.AddToxic("latency_down", "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": 200,
	})
	s.NoError(err)
	defer func() {
		err := proxy.RemoveToxic("latency_down")
		s.NoError(err)
	}()

	ctx := context.Background()
	loggerMock := logMocks.NewLogger(s.T())
	loggerMock.EXPECT().WithContext(matcher.Context).Return(loggerMock)
	loggerMock.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(loggerMock)
	loggerMock.EXPECT().Info("created new %s client %s", "dynamodb", "max_elapsed_time_exceeded").Once()

	client, err := gosoDdb.NewClient(ctx, s.Env().Config(), loggerMock, "max_elapsed_time_exceeded")
	s.NoError(err)

	_, err = client.PutItem(ctx, &awsDdb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{
				Value: "goso-id",
			},
		},
		TableName: aws.String("gosoline-cloud-dynamodb-test"),
	})

	s.True(exec.IsErrMaxElapsedTimeExceeded(err))
}

func (s *ClientTestSuite) TestRetryOnTransactionConflict() {
	ctx := context.Background()
	resource := &exec.ExecutableResource{
		Type: "DynamoDB",
		Name: "PutItem",
	}

	logger := logMocks.NewLogger(s.T())
	logger.EXPECT().WithContext(matcher.Context).Return(logger)
	logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(logger)
	logger.EXPECT().
		Warn(
			"attempt number %d to request resource %s failed after %s cause of error: %s",
			mock.AnythingOfType("int"),
			resource,
			mock.AnythingOfType("time.Duration"),
			mock.AnythingOfType("*types.TransactionCanceledException"),
		).Once()
	logger.EXPECT().Warn("sent request to resource %s successful after %d attempts in %s", resource, 2, mock.AnythingOfType("time.Duration")).Once()
	logger.EXPECT().Info("created new %s client %s", "dynamodb", "retryOnTransactionConflict").Once()

	client, err := gosoDdb.NewClient(ctx, s.Env().Config(), logger, "retryOnTransactionConflict")
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

			return stack.Deserialize.Add(middleware.DeserializeMiddlewareFunc(
				"injectTransactionConflict",
				func(ctx context.Context, input middleware.DeserializeInput, next middleware.DeserializeHandler) (middleware.DeserializeOutput, middleware.Metadata, error) {
					i++

					out, meta, err := next.HandleDeserialize(ctx, input)
					if err != nil {
						return out, meta, err
					}

					if i == 1 {
						err = &types.TransactionCanceledException{
							Message: aws.String(""),
							CancellationReasons: []types.CancellationReason{
								{
									Code: aws.String("TransactionConflict"),
								},
							},
						}
					}

					return out, meta, err
				},
			),
				middleware.Before,
			)
		})
	})

	s.NoError(err)
	logger.AssertExpectations(s.T())
}
