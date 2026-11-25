package consumer_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/justtrackio/gosoline/pkg/log"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

const (
	groupTypeTestTopic = "testapp-events"
	groupTypeTestGroup = "testapp-workers"
)

func TestGroupTypeValidation(t *testing.T) {
	tests := map[string]func(settings *consumer.Settings){
		"consumer group with round-robin balancer": func(settings *consumer.Settings) {
			settings.GroupType = consumer.GroupTypeConsumer
			settings.Balancers = []consumer.Balancer{consumer.RoundRobinBalancer}
		},
		"share group with ordered processing": func(settings *consumer.Settings) {
			settings.GroupType = consumer.GroupTypeShare
			settings.ProcessingMode = consumer.ProcessingModeOrdered
		},
		"share group with consume delay": func(settings *consumer.Settings) {
			settings.GroupType = consumer.GroupTypeShare
			settings.ConsumeDelay = time.Second
		},
		"unknown group type": func(settings *consumer.Settings) {
			settings.GroupType = "streams"
		},
		"share group without idle wait": func(settings *consumer.Settings) {
			settings.GroupType = consumer.GroupTypeShare
			settings.IdleWaitTime = 0
		},
		"share group without poll limit": func(settings *consumer.Settings) {
			settings.GroupType = consumer.GroupTypeShare
			settings.MaxPollRecords = 0
		},
	}

	for name, adjust := range tests {
		t.Run(name, func(t *testing.T) {
			settings := groupTypeTestSettings(consumer.GroupTypeConsumer)
			adjust(settings)

			_, err := consumer.NewReader(t.Context(), groupTypeTestConfig("localhost:9092"), log.NewLogger(), settings, nil, "test-consumer")
			assert.ErrorContains(t, err, "invalid kafka consumer settings")
		})
	}
}

// TestGroupTypeConsumer runs a consumer with the KIP-848 protocol and checks that the broker sees a "consumer" group
// which got the processed offsets committed.
func TestGroupTypeConsumer(t *testing.T) {
	cluster := newGroupTypeTestCluster(t, 2)
	produceGroupTypeTestRecords(t, cluster, 10)

	settings := groupTypeTestSettings(consumer.GroupTypeConsumer)
	cons := newGroupTypeTestConsumer(t, cluster, settings)

	var lck sync.Mutex
	received := 0

	runGroupTypeTestConsumer(t, cons, func(_ context.Context, _ *kgo.Record) bool {
		lck.Lock()
		defer lck.Unlock()

		received++
		if received == 10 {
			cons.Stop(context.Background())
		}

		return true
	})

	assert.Equal(t, 10, received)

	admin := newGroupTypeTestAdmin(t, cluster)

	groups, err := admin.ListGroups(t.Context())
	require.NoError(t, err)
	require.Contains(t, groups, groupTypeTestGroup)
	assert.Equal(t, "consumer", groups[groupTypeTestGroup].GroupType)

	offsets, err := admin.FetchOffsets(t.Context(), groupTypeTestGroup)
	require.NoError(t, err)

	committed := int64(0)
	offsets.Each(func(offset kadm.OffsetResponse) {
		committed += offset.At
	})
	assert.Equal(t, int64(10), committed, "the offsets of all processed records should be committed")
}

// TestGroupTypeShareRedeliversFailedRecords checks that a record whose processing failed is released and delivered
// again, while records which were processed successfully are accepted and delivered once.
func TestGroupTypeShareRedeliversFailedRecords(t *testing.T) {
	cluster := newGroupTypeTestCluster(t, 1)
	setShareGroupStartAtEarliest(t, cluster)
	produceGroupTypeTestRecords(t, cluster, 3)

	settings := groupTypeTestSettings(consumer.GroupTypeShare)
	cons := newGroupTypeTestConsumer(t, cluster, settings)

	var lck sync.Mutex
	deliveries := map[string][]int32{}
	processed := 0

	runGroupTypeTestConsumer(t, cons, func(_ context.Context, record *kgo.Record) bool {
		lck.Lock()
		defer lck.Unlock()

		value := string(record.Value)
		deliveries[value] = append(deliveries[value], record.DeliveryCount())

		if value == "record-1" && record.DeliveryCount() == 1 {
			return false
		}

		processed++
		if processed == 3 {
			cons.Stop(context.Background())
		}

		return true
	})

	assert.Equal(t, map[string][]int32{
		"record-0": {1},
		"record-1": {1, 2},
		"record-2": {1},
	}, deliveries)

	groups, err := newGroupTypeTestAdmin(t, cluster).ListGroups(t.Context())
	require.NoError(t, err)
	require.Contains(t, groups, groupTypeTestGroup)
	assert.Equal(t, "share", groups[groupTypeTestGroup].GroupType)
}

// TestGroupTypeShareConsumersShareAPartition runs two share group consumers on a topic with a single partition. Each
// one blocks processing its first record until the other one received a record as well, which only succeeds if both
// consume the same partition at the same time and neither of them locks records it does not process yet.
func TestGroupTypeShareConsumersShareAPartition(t *testing.T) {
	cluster := newGroupTypeTestCluster(t, 1)
	setShareGroupStartAtEarliest(t, cluster)
	produceGroupTypeTestRecords(t, cluster, 2)

	settings := groupTypeTestSettings(consumer.GroupTypeShare)
	settings.MaxPollRecords = 1

	first := newGroupTypeTestConsumer(t, cluster, settings)
	second := newGroupTypeTestConsumer(t, cluster, settings)

	var arrived sync.WaitGroup
	arrived.Add(2)
	bothArrived := make(chan struct{})

	go func() {
		arrived.Wait()
		close(bothArrived)
	}()

	process := func(cons consumer.Consumer) func(ctx context.Context, record *kgo.Record) bool {
		var once sync.Once

		return func(_ context.Context, _ *kgo.Record) bool {
			once.Do(arrived.Done)

			select {
			case <-bothArrived:
			case <-time.After(10 * time.Second):
				t.Error("the other consumer did not receive a record from the shared partition")
			}

			cons.Stop(context.Background())

			return true
		}
	}

	var wg sync.WaitGroup
	for _, cons := range []consumer.Consumer{first, second} {
		wg.Go(func() {
			runGroupTypeTestConsumer(t, cons, process(cons))
		})
	}
	wg.Wait()
}

func groupTypeTestSettings(groupType consumer.GroupType) *consumer.Settings {
	return &consumer.Settings{
		Connection:       "default",
		TopicId:          "events",
		GroupId:          "workers",
		GroupType:        groupType,
		StartOffset:      consumer.FirstOffset,
		Balancers:        []consumer.Balancer{consumer.CooperativeSticky},
		MaxPollRecords:   100,
		RebalanceTimeout: 10 * time.Second,
		SessionTimeout:   10 * time.Second,
		IdleWaitTime:     500 * time.Millisecond,
		GraceTime:        5 * time.Second,
		RunnerCount:      2,
		ProcessingMode:   consumer.ProcessingModeUnordered,
	}
}

func groupTypeTestConfig(broker string) cfg.Config {
	return cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "testapp",
		},
		"kafka": map[string]any{
			"naming": map[string]any{
				"topic_pattern": "{app.name}-{topicId}",
				"group_pattern": "{app.name}-{groupId}",
			},
			"connection": map[string]any{
				"default": map[string]any{
					"brokers":     []any{broker},
					"tls_enabled": false,
				},
			},
		},
	})
}

func newGroupTypeTestCluster(t *testing.T, partitions int32) *kfake.Cluster {
	cluster, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(partitions, groupTypeTestTopic))
	require.NoError(t, err)
	t.Cleanup(cluster.Close)

	return cluster
}

func newGroupTypeTestClient(t *testing.T, cluster *kfake.Cluster) *kgo.Client {
	client, err := kgo.NewClient(kgo.SeedBrokers(cluster.ListenAddrs()...), kgo.DefaultProduceTopic(groupTypeTestTopic))
	require.NoError(t, err)
	t.Cleanup(client.Close)

	return client
}

func newGroupTypeTestAdmin(t *testing.T, cluster *kfake.Cluster) *kadm.Client {
	return kadm.NewClient(newGroupTypeTestClient(t, cluster))
}

func produceGroupTypeTestRecords(t *testing.T, cluster *kfake.Cluster, count int) {
	client := newGroupTypeTestClient(t, cluster)

	for i := range count {
		require.NoError(t, client.ProduceSync(t.Context(), kgo.StringRecord(fmt.Sprintf("record-%d", i))).FirstErr())
	}
}

// setShareGroupStartAtEarliest makes the share group start at the beginning of the topic. Share groups ignore the
// client's start offset and default to the latest offset, which would skip the records the tests produce upfront.
func setShareGroupStartAtEarliest(t *testing.T, cluster *kfake.Cluster) {
	resource := kmsg.NewIncrementalAlterConfigsRequestResource()
	resource.ResourceType = kmsg.ConfigResourceTypeGroupConfig
	resource.ResourceName = groupTypeTestGroup

	config := kmsg.NewIncrementalAlterConfigsRequestResourceConfig()
	config.Name = "share.auto.offset.reset"
	config.Value = kmsg.StringPtr("earliest")
	resource.Configs = append(resource.Configs, config)

	req := kmsg.NewPtrIncrementalAlterConfigsRequest()
	req.Resources = append(req.Resources, resource)

	resp, err := req.RequestWith(t.Context(), newGroupTypeTestClient(t, cluster))
	require.NoError(t, err)

	for _, res := range resp.Resources {
		require.NoError(t, kerr.ErrorForCode(res.ErrorCode))
	}
}

func newGroupTypeTestConsumer(t *testing.T, cluster *kfake.Cluster, settings *consumer.Settings) consumer.Consumer {
	config := groupTypeTestConfig(cluster.ListenAddrs()[0])
	logger := log.NewLogger()

	healthCheckTimer, err := clock.NewHealthCheckTimer(time.Minute)
	require.NoError(t, err)

	metricWriter := metricMocks.NewWriter(t)
	metricWriter.On("Write", mock.Anything, mock.Anything).Maybe()
	metricWriter.On("WriteOne", mock.Anything, mock.Anything).Maybe()

	readerFactory := func(ctx context.Context, partitionManager *consumer.PartitionManager) (consumer.Reader, error) {
		return consumer.NewReader(ctx, config, logger, settings, partitionManager, "test-consumer")
	}

	return consumer.NewConsumerWithInterfaces(logger, clock.Provider, healthCheckTimer, readerFactory, settings, metricWriter, groupTypeTestTopic, "test-consumer")
}

// runGroupTypeTestConsumer runs the consumer until process stops it, failing the test if that takes too long.
func runGroupTypeTestConsumer(t *testing.T, cons consumer.Consumer, process func(ctx context.Context, record *kgo.Record) bool) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	require.NoError(t, cons.Run(ctx, process))
	require.NoError(t, ctx.Err(), "the consumer should have been stopped before the timeout")
}
