package oauth2

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	AuthTokenUrl  = "https://accounts.google.com/o/oauth2/token"
	TokenCheckUrl = "https://www.googleapis.com/oauth2/v3/tokeninfo?access_token=%s"
)

type GoogleTokenInfoResponse struct {
	AccessType string `json:"access_type"`
	Aud        string `json:"aud"`
	Azp        string `json:"azp"`
	Exp        string `json:"exp"`
	ExpiresIn  string `json:"expires_in"`
	Scope      string `json:"scope"`
}

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

//go:generate go run github.com/vektra/mockery/v2 --name Service
type Service interface {
	GetAuthRefresh(ctx context.Context, authRequest *GoogleAuthRequest) (*GoogleAuthResponse, error)
	TokenInfo(ctx context.Context, accessToken string) (*GoogleTokenInfoResponse, error)
}

type GoogleService struct {
	httpClient http.Client
}

func NewGoogleService(ctx context.Context, config cfg.Config, logger log.Logger) (Service, error) {
	httpClient, err := http.ProvideHttpClient(ctx, config, logger, "oauthGoogleService")
	if err != nil {
		return nil, fmt.Errorf("can not create http client: %w", err)
	}

	return NewGoogleServiceWithInterfaces(httpClient), nil
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

func (service *GoogleService) TokenInfo(ctx context.Context, accessToken string) (*GoogleTokenInfoResponse, error) {
	url := fmt.Sprintf(TokenCheckUrl, accessToken)

	request := service.httpClient.NewRequest().
		WithUrl(url)

	response, err := service.httpClient.Get(ctx, request)
	if err != nil {
		return nil, err
	}

	tokenInfoResponse := &GoogleTokenInfoResponse{}
	err = json.Unmarshal(response.Body, tokenInfoResponse)

	return tokenInfoResponse, err
}
