package auth

import "time"

type (
	Settings struct {
		AllowedAuthenticators []string `cfg:"allowedAuthenticators"`
	}

	JwtTokenHandlerSettings struct {
		SigningSecret  string        `cfg:"signingSecret"  validate:"min=8"`
		Issuer         string        `cfg:"issuer"         validate:"required"`
		ExpireDuration time.Duration `cfg:"expireDuration" validate:"min=60000000000" default:"15m"`
	}
)
