package stream_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	kinesisMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/suite"
)

type OutputKinesisTestSuite struct {
	suite.Suite

	ctx          context.Context
	recordWriter *kinesisMocks.RecordWriter
	output       stream.Output
}

func (s *OutputKinesisTestSuite) SetupSuite() {
	s.ctx = s.T().Context()
	s.recordWriter = kinesisMocks.NewRecordWriter(s.T())
	s.output = stream.NewKinesisOutputWithInterfaces(s.recordWriter)
}

func (s *OutputKinesisTestSuite) TestRawMessageSuccess() {
	expectedRecords := []*kinesis.Record{
		{
			Data: []byte(`"body"`),
		},
	}
	s.recordWriter.EXPECT().PutRecords(s.ctx, expectedRecords).Return(nil)

	rawMessage := stream.NewRawJsonMessage("body")

	err := s.output.WriteOne(s.ctx, rawMessage)
	s.NoError(err)
}

func (s *OutputKinesisTestSuite) TestStreamMessageSuccess() {
	expectedRecords := []*kinesis.Record{
		{
			Data:         []byte(`{"attributes":{"encoding":"application/json","gosoline.kinesis.partitionKey":"bfe5dfdc-0af2-44e5-863d-2c4860cc46d8"},"body":"body"}`),
			PartitionKey: aws.String("bfe5dfdc-0af2-44e5-863d-2c4860cc46d8"),
		},
	}
	s.recordWriter.EXPECT().PutRecords(s.ctx, expectedRecords).Return(nil)

	streamMessage := stream.NewJsonMessage("body", map[string]string{
		stream.AttributeKinesisPartitionKey: "bfe5dfdc-0af2-44e5-863d-2c4860cc46d8",
	})

	err := s.output.WriteOne(s.ctx, streamMessage)
	s.NoError(err)
}

func TestOutputKinesisTestSuite(t *testing.T) {
	suite.Run(t, new(OutputKinesisTestSuite))
}
