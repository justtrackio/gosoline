package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

//go:generate mockery --name JwtTokenHandler
type JwtTokenHandler interface {
	Sign(user SignUserInput) (*string, error)
	Valid(jwtToken string) (bool, *jwt.Token, error)
}

type jwtTokenHandler struct {
	settings JwtTokenHandlerSettings
}

type JwtTokenHandlerSettings struct {
	SigningSecret  string        `cfg:"signingSecret" validate:"min=8"`
	Issuer         string        `cfg:"issuer" validate:"required"`
	ExpireDuration time.Duration `cfg:"expireDuration" default:"15m" validate:"min=60000000000"`
}

type SignUserInput struct {
	Name  string
	Email string
	Image string
}

type JwtClaims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Image string `json:"image"`
	jwt.StandardClaims
}

func NewJwtTokenHandler(config cfg.Config, name string) JwtTokenHandler {
	key := fmt.Sprintf("httpserver.%s.auth.jwt", name)
	settings := &JwtTokenHandlerSettings{}
	config.UnmarshalKey(key, settings)

	return NewJwtTokenHandlerWithInterfaces(*settings)
}

func NewJwtTokenHandlerWithInterfaces(settings JwtTokenHandlerSettings) JwtTokenHandler {
	return &jwtTokenHandler{
		settings: settings,
	}
}

func (h *jwtTokenHandler) Sign(user SignUserInput) (*string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &JwtClaims{
		Name:  user.Name,
		Email: user.Email,
		Image: user.Image,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(h.settings.ExpireDuration).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    h.settings.Issuer,
		},
	})

	tokenString, err := token.SignedString([]byte(h.settings.SigningSecret))
	if err != nil {
		return nil, fmt.Errorf("could not sign jwt token: %w", err)
	}

	return &tokenString, nil
}

func (h *jwtTokenHandler) Valid(jwtToken string) (bool, *jwt.Token, error) {
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(h.settings.SigningSecret), nil
	})
	if err != nil {
		return false, nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if !claims.VerifyIssuer(h.settings.Issuer, true) {
			return false, nil, fmt.Errorf("invalid issuer")
		}

		if err := claims.Valid(); err != nil {
			return false, nil, fmt.Errorf("could not validate claims")
		}

		return true, token, nil
	}

	return false, nil, nil
}
