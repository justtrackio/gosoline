package consumer

import (
	"errors"

	"github.com/justtrackio/gosoline/pkg/exec"
	kafkaErrors "github.com/justtrackio/gosoline/pkg/kafka/errors"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kerr"
)

func newExecutorPolling(logger log.Logger, allowRebalance func(), settings *Settings) exec.Executor {
	res := &exec.ExecutableResource{
		Type: "kafka",
		Name: settings.TopicId,
	}

	return exec.NewBackoffExecutor(
		logger,
		res,
		&settings.Backoff,
		[]exec.ErrorChecker{
			CheckKafkaUnknownTopicError,
			checkKafkaRetryableErrorWithRebalance(allowRebalance),
		},
		exec.WithElapsedTimeTrackerFactory(func() exec.ElapsedTimeTracker {
			return exec.NewErrorTriggeredElapsedTimeTracker()
		}),
	)
}

func newExecutorCommitting(logger log.Logger, settings *Settings) exec.Executor {
	res := &exec.ExecutableResource{
		Type: "kafka",
		Name: settings.TopicId,
	}

	return exec.NewBackoffExecutor(
		logger,
		res,
		&exec.BackoffSettings{
			CancelDelay:     0, // use no cancel delay as we're using WithDelayedCancelContext
			InitialInterval: settings.Backoff.InitialInterval,
			MaxAttempts:     settings.Backoff.MaxAttempts,
			MaxElapsedTime:  settings.Backoff.MaxElapsedTime,
			MaxInterval:     settings.Backoff.MaxInterval,
		},
		[]exec.ErrorChecker{
			CheckKafkaRetryableError,
		},
		exec.WithElapsedTimeTrackerFactory(func() exec.ElapsedTimeTracker {
			return exec.NewErrorTriggeredElapsedTimeTracker()
		}),
	)
}

// CheckKafkaUnknownTopicError is an exec.ErrorChecker that fails fast when the consumer is configured
// for a topic that does not exist. franz-go surfaces these as UNKNOWN_TOPIC_OR_PARTITION (or
// UNKNOWN_TOPIC_ID) once KeepRetryableFetchErrors is enabled.
func CheckKafkaUnknownTopicError(_ any, err error) exec.ErrorType {
	if isUnknownTopicError(err) {
		return exec.ErrorTypePermanent
	}

	return exec.ErrorTypeUnknown
}

func isUnknownTopicError(err error) bool {
	return errors.Is(err, kerr.UnknownTopicOrPartition) || errors.Is(err, kerr.UnknownTopicID)
}

func checkKafkaRetryableErrorWithRebalance(allowRebalance func()) func(_ any, err error) exec.ErrorType {
	return func(_ any, err error) exec.ErrorType {
		switch {
		case err == nil:
			return exec.ErrorTypeOk
		case kafkaErrors.IsRetryableKafkaError(err):
			allowRebalance()

			return exec.ErrorTypeRetryable
		default:
			return exec.ErrorTypeUnknown
		}
	}
}

func CheckKafkaRetryableError(_ any, err error) exec.ErrorType {
	switch {
	case err == nil:
		return exec.ErrorTypeOk
	case kafkaErrors.IsRetryableKafkaError(err):
		return exec.ErrorTypeRetryable
	default:
		return exec.ErrorTypeUnknown
	}
}
