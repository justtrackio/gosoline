package kinesis

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/log"
	"github.com/twinj/uuid"
	"github.com/twitchscience/kinsumer"
	"time"
)

//go:generate mockery -name Kinsumer
type Kinsumer interface {
	Run() error
	Next() (data []byte, err error)
	Stop()
}

type kinsumerLogger struct {
	logger log.Logger
}

func (k kinsumerLogger) Log(format string, args ...interface{}) {
	k.logger.Info(format, args...)
}

type KinsumerSettings struct {
	StreamName      string
	ApplicationName string
}

func (k *KinsumerSettings) GetResourceName() string {
	return k.StreamName
}

func NewKinsumer(config cfg.Config, logger log.Logger, settings KinsumerSettings) (Kinsumer, error) {
	kinesisClient := cloud.GetKinesisClient(config, logger)
	dynamoDbClient := cloud.GetDynamoDbClient(config, logger)

	clientName := uuid.NewV4().String()

	logger = logger.WithFields(log.Fields{
		"applicationName":  settings.ApplicationName,
		"clientIdentifier": clientName,
		"inputStream":      settings.StreamName,
	}).WithChannel("kinsumer")

	err := CreateKinesisStream(config, logger, kinesisClient, &settings)

	if err != nil {
		return nil, fmt.Errorf("failed to create kinesis stream: %w", err)
	}
	logger.Info("starting stream reader")

	shardCheckFreq := config.GetDuration("aws_kinesis_shard_check_freq") * time.Second
	leaderActionFreq := config.GetDuration("aws_kinesis_leader_action_freq") * time.Second

	kinsumerConfig := kinsumer.NewConfig()
	kinsumerConfig.WithShardCheckFrequency(shardCheckFreq)
	kinsumerConfig.WithLeaderActionFrequency(leaderActionFreq)
	kinsumerConfig.WithLogger(kinsumerLogger{
		logger: logger,
	})

	client, err := kinsumer.NewWithInterfaces(kinesisClient, dynamoDbClient, settings.StreamName, settings.ApplicationName, clientName, kinsumerConfig)

	if err != nil {
		return nil, fmt.Errorf("error creating kinsumer: %w", err)
	}

	err = client.CreateRequiredTables()

	if err != nil {
		return nil, fmt.Errorf("error creating kinsumer dynamo db tables: %w", err)
	}

	return client, nil
}
