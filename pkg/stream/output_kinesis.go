package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
	gosoKinesis "github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/spf13/cast"
)

const (
	AttributeKinesisPartitionKey    = "gosoline.kinesis.partitionKey"
	AttributeKinesisExplicitHashKey = "gosoline.kinesis.explicitHashKey"
)

type KinesisOutputSettings struct {
	cfg.AppId
	ClientName string
	StreamName string
}

func (s KinesisOutputSettings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s KinesisOutputSettings) GetClientName() string {
	return s.ClientName
}

func (s KinesisOutputSettings) GetStreamName() string {
	return s.StreamName
}

type kinesisOutput struct {
	recordWriter gosoKinesis.RecordWriter
}

func NewKinesisOutput(ctx context.Context, config cfg.Config, logger log.Logger, settings *KinesisOutputSettings) (Output, error) {
	settings.PadFromConfig(config)

	var err error
	var recordWriter gosoKinesis.RecordWriter

	clientsKey := aws.GetClientConfigKey("kinesis", settings.ClientName)
	defaultClientKey := aws.GetClientConfigKey("kinesis", "default")
	clientDefaultsKey := aws.GetDefaultsKey(settings.ClientName)
	defaultsKey := aws.GetDefaultsKey("default")

	backoffSettings, err := exec.ReadBackoffSettings(config, clientsKey, clientDefaultsKey, defaultClientKey, defaultsKey, "cloud.aws.defaults")
	if err != nil {
		return nil, fmt.Errorf("failed to read backoff settings for kinesis: %w", err)
	}
	backoffSettings.InitialInterval = time.Second

	recordWriterSettings := &gosoKinesis.RecordWriterSettings{
		AppId:      settings.AppId,
		ClientName: settings.ClientName,
		StreamName: settings.GetStreamName(),
		Backoff:    backoffSettings,
	}

	if recordWriter, err = gosoKinesis.NewRecordWriter(ctx, config, logger, recordWriterSettings); err != nil {
		return nil, fmt.Errorf("can not create record writer for stream %s: %w", settings.GetStreamName(), err)
	}

	return NewKinesisOutputWithInterfaces(recordWriter), nil
}

func NewKinesisOutputWithInterfaces(recordWriter gosoKinesis.RecordWriter) Output {
	return &kinesisOutput{
		recordWriter: recordWriter,
	}
}

func (o *kinesisOutput) WriteOne(ctx context.Context, record WritableMessage) error {
	return o.Write(ctx, []WritableMessage{record})
}

func (o *kinesisOutput) Write(ctx context.Context, batch []WritableMessage) error {
	var err error
	records := make([]*gosoKinesis.Record, len(batch))

	for i, msg := range batch {
		if records[i], err = o.buildRecord(msg); err != nil {
			return fmt.Errorf("can not build record: %w", err)
		}
	}

	return o.recordWriter.PutRecords(ctx, records)
}

func (o *kinesisOutput) IsPartitionedOutput() bool {
	return true
}

func (o *kinesisOutput) GetMaxMessageSize() *int {
	return mdl.Box(1024 * 1024)
}

func (o *kinesisOutput) GetMaxBatchSize() *int {
	return mdl.Box(500)
}

func (o *kinesisOutput) buildRecord(msg WritableMessage) (*gosoKinesis.Record, error) {
	var err error
	var partitionKey, explicitHashKey string

	record := &gosoKinesis.Record{}

	if record.Data, err = msg.MarshalToBytes(); err != nil {
		return nil, fmt.Errorf("can not marshal message to bytes: %w", err)
	}

	attributes := getAttributes(msg)

	if p, ok := attributes[AttributeKinesisPartitionKey]; ok {
		if partitionKey, err = cast.ToStringE(p); err != nil {
			return nil, fmt.Errorf("the type of the %s attribute with value %v should be castable to string: %w", AttributeKinesisPartitionKey, attributes[AttributeKinesisPartitionKey], err)
		}

		record.PartitionKey = &partitionKey
	}

	if p, ok := attributes[AttributeKinesisExplicitHashKey]; ok {
		if explicitHashKey, err = cast.ToStringE(p); err != nil {
			return nil, fmt.Errorf("the type of the %s attribute with value %v should be castable to string: %w", AttributeKinesisExplicitHashKey, attributes[AttributeKinesisExplicitHashKey], err)
		}

		record.ExplicitHashKey = &explicitHashKey
		// yes, this looks strange, but we need to provide something or AWS complains, so we just do that
		// and hope it is ignored as documented...
		record.PartitionKey = &explicitHashKey
	}

	return record, nil
}
