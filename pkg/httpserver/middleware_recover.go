package httpserver

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

func RecoveryWithSentry(logger log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			err := recover()

			switch rval := err.(type) {
			case nil:
				return
			case error:
				if errors.Is(rval, ResponseBodyWriterError{}) && exec.IsConnectionError(rval) {
					logger.Warn("connection error: %s", rval.Error())
					return
				}

				logger.Error("%w", rval)
			case string:
				logger.Error(rval)
			default:
			}

			c.AbortWithStatus(http.StatusInternalServerError)
		}()

		c.Next()
	}
}
