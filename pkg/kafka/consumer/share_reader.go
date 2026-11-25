package consumer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

// shareReader adapts a franz-go share group client to the Reader the consumer drives.
//
// A share group has neither rebalances to allow nor offsets to commit. The consumer acknowledges every record it handed
// to the processing callback right after processing it (see consumer.acknowledge), so committing a batch only has to
// settle the records which were never handed over and to wait until the broker received all acknowledgements.
type shareReader struct {
	client *kgo.Client
}

func newShareReader(ctx context.Context, logger log.Logger, settings *Settings, groupId string, opts []kgo.Opt) (Reader, error) {
	opts = append(opts,
		kgo.ShareGroup(groupId),
		// Every record a share fetch returns is locked for this consumer until it is acknowledged or the lock
		// expires. Fetching only what one poll hands out keeps records from waiting, locked, in a client buffer.
		kgo.ShareMaxRecords(int32(settings.MaxPollRecords)),
		kgo.ShareMaxRecordsStrict(),
		kgo.ShareAckCallback(func(_ *kgo.Client, results kgo.ShareAckResults) {
			for _, result := range results {
				if result.Err == nil {
					continue
				}

				logger.Warn(ctx, "failed to acknowledge share group records (topic: %s, partition: %d), the broker delivers them again once their lock expires: %s",
					result.Topic, result.Partition, result.Err.Error())
			}
		}),
	)

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("can not create franz-go share group client: %w", err)
	}

	return &shareReader{client: client}, nil
}

// AllowRebalance is a no-op, a share group does not block assignment changes while records are processed.
func (r *shareReader) AllowRebalance() {}

// CloseAllowingRebalance leaves the share group. franz-go releases all records which are still unacknowledged.
func (r *shareReader) CloseAllowingRebalance() {
	r.client.Close()
}

// CommitRecords releases every record of the last poll that was not acknowledged yet, which are the records a shutdown
// kept from being processed, and waits until the broker received all acknowledgements. The given records were already
// acknowledged one by one.
//
// A record has to be settled before the next poll, as franz-go accepts every record of the previous poll that is still
// unacknowledged at that point.
func (r *shareReader) CommitRecords(ctx context.Context, _ ...*kgo.Record) error {
	r.client.MarkAcks(kgo.AckRelease)

	if err := r.client.FlushAcks(ctx); err != nil {
		return fmt.Errorf("failed to flush share group acknowledgements: %w", err)
	}

	return nil
}

func (r *shareReader) PollRecords(ctx context.Context, maxPollRecords int) kgo.Fetches {
	return r.client.PollRecords(ctx, maxPollRecords)
}
