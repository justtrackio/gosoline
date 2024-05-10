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
	Attributes      map[string]any
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

func OnlyConfiguredAuthenticators(config cfg.Config, name string, authenticators map[string]Authenticator) (map[string]Authenticator, error) {
	key := fmt.Sprintf("httpserver.%s.auth", name)
	settings := &Settings{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth settings: %w", err)
	}
	if len(settings.AllowedAuthenticators) == 0 {
		return authenticators, nil
	}

	return funk.IntersectMaps(authenticators, funk.SliceToMap(settings.AllowedAuthenticators, func(method string) (string, Authenticator) { return method, nil })), nil
}
