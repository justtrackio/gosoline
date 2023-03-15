//go:build integration

package cloudwatch_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsCw "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ClientTestSuite struct {
	suite.Suite
}

func (s *ClientTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithConfigFile("client_test_cfg.yml"),
		suite.WithLogLevel("debug"),
		suite.WithClockProvider(clock.NewRealClock()),
	}
}

func (s *ClientTestSuite) TestNewDefault() {
	client, err := cloudwatch.NewClient(context.Background(), s.Env().Config(), s.Env().Logger(), "default")
	s.NoError(err)

	_, err = client.GetMetricStatistics(context.Background(), &awsCw.GetMetricStatisticsInput{
		StartTime:  aws.Time(time.Now().Add(time.Hour * -1)),
		EndTime:    aws.Time(time.Now()),
		Namespace:  aws.String("gosoline"),
		MetricName: aws.String("test"),
		Period:     aws.Int32(60),
		Statistics: []types.Statistic{
			types.StatisticSum,
		},
	})
	s.NoError(err)
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
