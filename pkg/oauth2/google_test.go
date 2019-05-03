package oauth2

import (
	"errors"
	"github.com/applike/gosoline/pkg/http"
	httpMocks "github.com/applike/gosoline/pkg/http/mocks"
	"github.com/gin-gonic/gin/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestGoogleService_GetAuthRefresh(t *testing.T) {
	googleAuthRequest := &GoogleAuthRequest{
		ClientId:     "ClientId",
		ClientSecret: "ClientSecret",
		GrantType:    "GrantType",
		RefreshToken: "RefreshToken",
	}
	expectedGoogleAuthResponse := &GoogleAuthResponse{
		AccessToken: "at-at",
		ExpiresIn:   1,
		TokenType:   "grizzly",
	}
	httpRequest := http.NewRequest("https://accounts.google.com/o/oauth2/token")
	httpRequest.Body = map[string]string{
		"client_id":     googleAuthRequest.ClientId,
		"client_secret": googleAuthRequest.ClientSecret,
		"grant_type":    googleAuthRequest.GrantType,
		"refresh_token": googleAuthRequest.RefreshToken,
	}
	httpResponse, err := json.Marshal(expectedGoogleAuthResponse)

	assert.NoError(t, err)

	httpClient := new(httpMocks.Client)
	httpClient.On("Post", httpRequest).Return(httpResponse, nil)

	service := NewGoogleServiceWithInterfaces(httpClient)
	googleAuthResponse, err := service.GetAuthRefresh(googleAuthRequest)

	assert.NoError(t, err)
	assert.Equal(t, expectedGoogleAuthResponse.AccessToken, googleAuthResponse.AccessToken)
	assert.Equal(t, expectedGoogleAuthResponse.TokenType, googleAuthResponse.TokenType)
	assert.Equal(t, expectedGoogleAuthResponse.ExpiresIn, googleAuthResponse.ExpiresIn)

	httpClient.AssertExpectations(t)
}

func TestGoogleService_GetAuthRefresh_Error(t *testing.T) {
	googleAuthRequest := &GoogleAuthRequest{
		ClientId:     "ClientId",
		ClientSecret: "ClientSecret",
		GrantType:    "GrantType",
		RefreshToken: "RefreshToken",
	}

	httpClient := new(httpMocks.Client)
	httpClient.On("Post", mock.Anything).Return(nil, errors.New("test"))

	service := NewGoogleServiceWithInterfaces(httpClient)
	_, err := service.GetAuthRefresh(googleAuthRequest)

	assert.Error(t, err)
}
