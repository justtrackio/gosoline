package oauth2

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/log"
)

const AuthTokenUrl = "https://accounts.google.com/o/oauth2/token"

type GoogleAuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   uint   `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type GoogleAuthRequest struct {
	ClientId     string
	ClientSecret string
	GrantType    string
	RefreshToken string
}

//go:generate mockery -name Service
type Service interface {
	GetAuthRefresh(ctx context.Context, authRequest *GoogleAuthRequest) (*GoogleAuthResponse, error)
}

type GoogleService struct {
	httpClient http.Client
}

func NewGoogleService(config cfg.Config, logger log.Logger) Service {
	httpClient := http.NewHttpClient(config, logger)

	return NewGoogleServiceWithInterfaces(httpClient)
}

func NewGoogleServiceWithInterfaces(httpClient http.Client) Service {
	return &GoogleService{
		httpClient: httpClient,
	}
}

func (service *GoogleService) GetAuthRefresh(ctx context.Context, authRequest *GoogleAuthRequest) (*GoogleAuthResponse, error) {
	request := service.httpClient.NewRequest().
		WithUrl(AuthTokenUrl).
		WithBody(map[string]string{
			"client_id":     authRequest.ClientId,
			"client_secret": authRequest.ClientSecret,
			"grant_type":    authRequest.GrantType,
			"refresh_token": authRequest.RefreshToken,
		})

	response, err := service.httpClient.Post(ctx, request)

	if err != nil {
		return nil, err
	}

	authResponse := &GoogleAuthResponse{}
	err = json.Unmarshal(response.Body, authResponse)

	return authResponse, err
}
