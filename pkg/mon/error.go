package mon

import "fmt"

func WrapErrorAndLog(logger Logger, err error, msg string, args ...interface{}) error {
	errMsg := fmt.Sprintf(msg, args...)
	err = fmt.Errorf("%s: %w", errMsg, err)

	logger.Error(err.Error())

	return err
}
