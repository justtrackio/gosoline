package mon

import "fmt"

func WrapErrorAndLog(logger Logger, err error, msg string, args ...interface{}) error {
	errMsg := fmt.Sprintf(msg, args...)
	logger.Error(err, errMsg)

	return fmt.Errorf("%s: %w", errMsg, err)
}
