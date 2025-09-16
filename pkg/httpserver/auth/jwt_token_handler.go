package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

//go:generate go run github.com/vektra/mockery/v2 --name JwtTokenHandler
type JwtTokenHandler interface {
	Sign(user SignUserInput) (*string, error)
	SignClaims(claims Claims) (*string, error)
	Valid(jwtToken string) (bool, *jwt.Token, error)
}

type Claims interface {
	jwt.Claims
	GetRegisteredClaims() jwt.RegisteredClaims
	SetRegisteredClaims(registeredClaims jwt.RegisteredClaims)
}

type jwtTokenHandler struct {
	settings JwtTokenHandlerSettings
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
	jwt.RegisteredClaims
}

func (c *JwtClaims) GetRegisteredClaims() jwt.RegisteredClaims {
	return c.RegisteredClaims
}

func (c *JwtClaims) SetRegisteredClaims(registeredClaims jwt.RegisteredClaims) {
	c.RegisteredClaims = registeredClaims
}

func NewJwtTokenHandler(config cfg.Config, name string) (JwtTokenHandler, error) {
	key := fmt.Sprintf("httpserver.%s.auth.jwt", name)
	settings := &JwtTokenHandlerSettings{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal jwt token handler settings: %w", err)
	}

	return NewJwtTokenHandlerWithInterfaces(*settings), nil
}

func NewJwtTokenHandlerWithInterfaces(settings JwtTokenHandlerSettings) JwtTokenHandler {
	return &jwtTokenHandler{
		settings: settings,
	}
}

func (h *jwtTokenHandler) Sign(user SignUserInput) (*string, error) {
	return h.SignClaims(&JwtClaims{
		Name:  user.Name,
		Email: user.Email,
		Image: user.Image,
	})
}

func (h *jwtTokenHandler) SignClaims(claims Claims) (*string, error) {
	registeredClaims := claims.GetRegisteredClaims()
	registeredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(h.settings.ExpireDuration))
	registeredClaims.IssuedAt = jwt.NewNumericDate(time.Now())
	registeredClaims.Issuer = h.settings.Issuer
	claims.SetRegisteredClaims(registeredClaims)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(h.settings.SigningSecret))
	if err != nil {
		return nil, fmt.Errorf("could not sign jwt token: %w", err)
	}

	return &tokenString, nil
}

func (h *jwtTokenHandler) Valid(jwtToken string) (bool, *jwt.Token, error) {
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(h.settings.SigningSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return false, nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		issuer, err := claims.GetIssuer()
		if err != nil {
			return false, nil, fmt.Errorf("could not get issuer: %w", err)
		}
		if issuer != h.settings.Issuer {
			return false, nil, fmt.Errorf("invalid issuer")
		}

		return true, token, nil
	}

	return false, nil, nil
}
