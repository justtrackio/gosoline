package auth

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func NewChainHandler(authenticators map[string]Authenticator) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		errors := make(map[string]string)

		for n, a := range authenticators {
			valid, err := a.IsValid(ginCtx)

			if err != nil {
				errors[n] = err.Error()
				continue
			}

			if valid {
				return
			}
		}

		ginCtx.JSON(http.StatusUnauthorized, errors)
		ginCtx.Abort()
	}
}
