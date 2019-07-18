package oauth2

import (
	"encoding/json"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/mon"
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
	GetAuthRefresh(refreshToken string) *GoogleAuthResponse
}

type GoogleService struct {
	httpClient http.Client
}

func NewGoogleService(logger mon.Logger) *GoogleService {
	httpClient := http.NewHttpClient(logger)

	return NewGoogleServiceWithInterfaces(httpClient)
}

func NewGoogleServiceWithInterfaces(httpClient http.Client) *GoogleService {
	return &GoogleService{
		httpClient: httpClient,
	}
}

func (service *GoogleService) GetAuthRefresh(authRequest *GoogleAuthRequest) (*GoogleAuthResponse, error) {
	request := http.NewRequest(AuthTokenUrl)
	request.Body = map[string]string{
		"client_id":     authRequest.ClientId,
		"client_secret": authRequest.ClientSecret,
		"grant_type":    authRequest.GrantType,
		"refresh_token": authRequest.RefreshToken,
	}

	response, err := service.httpClient.Post(request)

	if err != nil {
		return nil, err
	}

	authResponse := &GoogleAuthResponse{}
	err = json.Unmarshal(response.Body, authResponse)

	return authResponse, err
}
