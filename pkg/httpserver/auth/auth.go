package auth

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
)

const Anonymous = "anon"

//go:generate go run github.com/vektra/mockery/v2 --name Authenticator
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

func OnlyConfiguredAuthenticators(config cfg.Config, name string, authenticators map[string]Authenticator) map[string]Authenticator {
	key := fmt.Sprintf("httpserver.%s.auth", name)
	settings := &Settings{}
	config.UnmarshalKey(key, settings)
	if len(settings.AllowedAuthenticators) == 0 {
		return authenticators
	}

	return funk.IntersectMaps(authenticators, funk.SliceToMap(settings.AllowedAuthenticators, func(method string) (string, Authenticator) { return method, nil }))
}
