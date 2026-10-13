package auth

import (
	"reflect"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type (
	Settings struct {
		AllowedAuthenticators []string `cfg:"allowedAuthenticators"`
	}

	JwtTokenHandlerSettings struct {
		SigningSecret  string        `cfg:"signingSecret,secret" validate:"min=32"`
		Issuer         string        `cfg:"issuer"         validate:"required"`
		ExpireDuration time.Duration `cfg:"expireDuration" validate:"min=60000000000" default:"15m"`
	}
)

type httpserverDebugSettings struct {
	Auth jwtAuthDebugSettings `cfg:"auth"`
}

type jwtAuthDebugSettings struct {
	Jwt JwtTokenHandlerSettings `cfg:"jwt"`
}

func init() {
	cfg.RegisterConfigDebugType("httpserver", reflect.TypeFor[map[string]httpserverDebugSettings]())
}
