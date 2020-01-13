package mon

import "fmt"

func WrapErrorAndLog(logger Logger, err error, msg string, args ...interface{}) error {
	logger.Errorf(err, msg, args...)

	errMsg := fmt.Sprintf(msg, args...)
	err = fmt.Errorf("%s: %w", errMsg, err)

	return err
}
