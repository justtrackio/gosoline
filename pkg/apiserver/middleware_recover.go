package apiserver

import (
	"errors"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/gin-gonic/gin"
	"net/http"
)

func RecoveryWithSentry(logger mon.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			err := recover()

			switch rval := err.(type) {
			case nil:
				return
			case error:
				logger.Error(rval, rval.Error())
			case string:
				logger.Error(errors.New(err.(string)), err.(string))
			default:
			}

			c.AbortWithStatus(http.StatusInternalServerError)
		}()

		c.Next()
	}
}
