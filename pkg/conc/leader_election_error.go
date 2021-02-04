package conc

import (
	"errors"
	"fmt"
)

func IsLeaderElectionFatalError(err error) bool {
	return errors.As(err, &LeaderElectionFatalError{})
}

type LeaderElectionFatalError struct {
	err error
}

func NewLeaderElectionFatalError(err error) LeaderElectionFatalError {
	return LeaderElectionFatalError{
		err: err,
	}
}

func (e LeaderElectionFatalError) Error() string {
	return fmt.Sprintf("fatal error during leader election: %s", e.err)
}

func (e LeaderElectionFatalError) Unwrap() error {
	return e.err
}

func IsLeaderElectionTransientError(err error) bool {
	return errors.As(err, &LeaderElectionTransientError{})
}

type LeaderElectionTransientError struct {
	err error
}

func NewLeaderElectionTransientError(err error) LeaderElectionTransientError {
	return LeaderElectionTransientError{
		err: err,
	}
}

func (e LeaderElectionTransientError) Error() string {
	return fmt.Sprintf("transient error during leader election: %s", e.err)
}

func (e LeaderElectionTransientError) Unwrap() error {
	return e.err
}
