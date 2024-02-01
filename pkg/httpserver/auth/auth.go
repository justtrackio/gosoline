package auth

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
)

const Anonymous = "anon"

//go:generate mockery --name Authenticator
type Authenticator interface {
	IsValid(ginCtx *gin.Context) (bool, error)
}

type subjectKeyType int

var subjectKey = new(subjectKeyType)

type Subject struct {
	Name            string
	Anonymous       bool
	AuthenticatedBy string
	Attributes      map[string]interface{}
}

func RequestWithSubject(ginCtx *gin.Context, subject *Subject) {
	reqCtx := ginCtx.Request.Context()
	newCtx := context.WithValue(reqCtx, subjectKey, subject)

	ginCtx.Request = ginCtx.Request.WithContext(newCtx)
}

func GetSubject(ctx context.Context) *Subject {
	if user, ok := ctx.Value(subjectKey).(*Subject); ok {
		return user
	}

	panic(fmt.Errorf("there is no subject in the context"))
}
