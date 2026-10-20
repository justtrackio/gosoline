package limit

import (
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/justtrackio/gosoline/pkg/clock"
	dynamodbMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb/mocks"
	"github.com/justtrackio/gosoline/pkg/ddb"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/require"
)

func TestFixedWindowDdbRenewsExpiry(t *testing.T) {
	base := time.Unix(1700000000, 0)
	for _, test := range []struct {
		name           string
		offset         time.Duration
		duration       time.Duration
		cleanupOffsets [2]int64
	}{
		{name: "nanosecond windows", offset: 100 * time.Millisecond, duration: time.Nanosecond, cleanupOffsets: [2]int64{1, 1}},
		{name: "half-second windows", offset: 100 * time.Millisecond, duration: 500 * time.Millisecond, cleanupOffsets: [2]int64{1, 2}},
		{name: "whole-second windows with fractional start", offset: 100 * time.Millisecond, duration: time.Second, cleanupOffsets: [2]int64{2, 3}},
		{name: "exact whole-second expiry", duration: 10 * time.Second, cleanupOffsets: [2]int64{10, 20}},
	} {
		t.Run(test.name, func(t *testing.T) {
			backend, client := newFixedWindowDdbTestBackend(t, base.Add(test.offset), test.duration)
			fakeClock := backend.clock.(clock.FakeClock)
			for i, cleanupOffset := range test.cleanupOffsets {
				now := fakeClock.Now()
				expiresAt := now.Add(test.duration)
				attributes, err := ddb.MarshalMap(&window{
					Key: "test/network", Val: 1,
					WindowExpiresAt: expiresAt.UnixNano(),
					Ttl:             base.Unix() + cleanupOffset,
				})
				require.NoError(t, err)
				client.EXPECT().UpdateItem(matcher.Context, fixedWindowDdbResetInput(now, test.duration)).Return(&dynamodb.UpdateItemOutput{Attributes: attributes}, nil).Once()

				count, remaining, err := backend.Increment(t.Context(), "network")

				require.NoError(t, err)
				require.Equal(t, 1, *count, "window %d", i)
				require.Equal(t, test.duration, *remaining, "window %d", i)
				fakeClock.Advance(test.duration)
			}
		})
	}
}

func TestFixedWindowDdbIncrementsActiveWindow(t *testing.T) {
	base := time.Unix(1700000000, 0)
	for _, test := range []struct {
		name      string
		now       time.Time
		item      window
		remaining time.Duration
	}{
		{
			name: "precise window", now: base.Add(200 * time.Millisecond),
			item:      window{Key: "test/network", Val: 2, Ttl: base.Unix() + 1, WindowExpiresAt: base.Add(600 * time.Millisecond).UnixNano()},
			remaining: 400 * time.Millisecond,
		},
		{
			name: "legacy row without precise expiry", now: base.Add(200 * time.Millisecond),
			item:      window{Key: "test/network", Val: 2, Ttl: base.Unix() + 1},
			remaining: 1800 * time.Millisecond,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			backend, client := newFixedWindowDdbTestBackend(t, test.now, 500*time.Millisecond)
			client.EXPECT().UpdateItem(matcher.Context, fixedWindowDdbResetInput(test.now, 500*time.Millisecond)).Return(nil, &types.ConditionalCheckFailedException{}).Once()
			attributes, err := ddb.MarshalMap(&test.item)
			require.NoError(t, err)
			client.EXPECT().UpdateItem(matcher.Context, &dynamodb.UpdateItemInput{
				TableName:                 aws.String("rate-limits"),
				Key:                       map[string]types.AttributeValue{"key": &types.AttributeValueMemberS{Value: "test/network"}},
				ExpressionAttributeNames:  map[string]string{"#0": "val"},
				ExpressionAttributeValues: map[string]types.AttributeValue{":0": &types.AttributeValueMemberN{Value: "1"}},
				UpdateExpression:          aws.String("ADD #0 :0\n"),
				ReturnValues:              types.ReturnValueAllNew,
				ReturnConsumedCapacity:    types.ReturnConsumedCapacityIndexes,
			}).Return(&dynamodb.UpdateItemOutput{Attributes: attributes}, nil).Once()

			count, remaining, err := backend.Increment(t.Context(), "network")

			require.NoError(t, err)
			require.Equal(t, 2, *count)
			require.Equal(t, test.remaining, *remaining)
		})
	}
}

func newFixedWindowDdbTestBackend(t *testing.T, now time.Time, duration time.Duration) (*fixedWindowDdb, *dynamodbMocks.Client) {
	t.Helper()
	client := dynamodbMocks.NewClient(t)
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	factory := ddb.NewMetadataFactoryWithInterfaces(&ddb.Settings{Main: ddb.MainSettings{Model: &window{}}}, "rate-limits")
	repo, err := ddb.NewWithInterfaces(logger, tracing.NewLocalTracer(), client, factory)
	require.NoError(t, err)

	return &fixedWindowDdb{repo: repo, clock: clock.NewFakeClockAt(now), name: "test", window: duration}, client
}

func fixedWindowDdbResetInput(now time.Time, duration time.Duration) *dynamodb.UpdateItemInput {
	expiresAt := now.Add(duration)
	// Rounding up must not delay cleanup when expiry is already a whole second.
	cleanupTtl := expiresAt.Add(time.Second - time.Nanosecond).Unix()

	return &dynamodb.UpdateItemInput{
		TableName:                aws.String("rate-limits"),
		Key:                      map[string]types.AttributeValue{"key": &types.AttributeValueMemberS{Value: "test/network"}},
		ConditionExpression:      aws.String("(#0 <= :0) OR ((attribute_not_exists (#0)) AND ((attribute_not_exists (#1)) OR (#1 < :1)))"),
		ExpressionAttributeNames: map[string]string{"#0": "windowExpiresAt", "#1": "ttl", "#2": "val"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":0": &types.AttributeValueMemberN{Value: strconv.FormatInt(now.UnixNano(), 10)},
			":1": &types.AttributeValueMemberN{Value: strconv.FormatInt(now.Unix(), 10)},
			":2": &types.AttributeValueMemberN{Value: "1"},
			":3": &types.AttributeValueMemberN{Value: strconv.FormatInt(expiresAt.UnixNano(), 10)},
			":4": &types.AttributeValueMemberN{Value: strconv.FormatInt(cleanupTtl, 10)},
		},
		UpdateExpression:       aws.String("SET #2 = :2, #0 = :3, #1 = :4\n"),
		ReturnValues:           types.ReturnValueAllNew,
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}
}
