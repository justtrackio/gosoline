package stream

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/twinj/uuid"
	"github.com/twitchscience/kinsumer"
	"time"
)

type Kinsumer interface {
	Run() error
	Next() (data []byte, err error)
	Stop()
}

type KinsumerFactory func(config cfg.Config, logger mon.Logger, settings KinsumerSettings) Kinsumer

func NewKinsumer(config cfg.Config, logger mon.Logger, settings KinsumerSettings) Kinsumer {
	kinesisClient := cloud.GetKinesisClient(config, logger)
	dynamoDbClient := cloud.GetDynamoDbClient(config, logger)

	createKinesisStream(config, logger, kinesisClient, &settings)

	clientName := uuid.NewV4().String()

	logger.WithFields(mon.Fields{
		"applicationName":  settings.ApplicationName,
		"clientIdentifier": clientName,
		"inputStream":      settings.StreamName,
	}).Info("starting stream reader")

	shardCheckFreq := config.GetDuration("aws_kinesis_shard_check_freq") * time.Second
	leaderActionFreq := config.GetDuration("aws_kinesis_leader_action_freq") * time.Second

	kinsumerConfig := kinsumer.NewConfig()
	kinsumerConfig.WithShardCheckFrequency(shardCheckFreq)
	kinsumerConfig.WithLeaderActionFrequency(leaderActionFreq)

	client, err := kinsumer.NewWithInterfaces(kinesisClient, dynamoDbClient, settings.StreamName, settings.ApplicationName, clientName, kinsumerConfig)

	if err != nil {
		logger.Fatal(err, "Error creating kinsumer")
	}

	err = client.CreateRequiredTables()

	if err != nil {
		logger.Fatal(err, "Error creating kinsumer dynamo db tables")
	}

	return client
}
