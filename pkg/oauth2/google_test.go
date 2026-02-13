package oauth2

import (
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/http"
	httpMocks "github.com/justtrackio/gosoline/pkg/http/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	httpRequest := http.NewRequest(nil).
		WithUrl("https://accounts.google.com/o/oauth2/token").
		WithBody(map[string]string{
			"client_id":     googleAuthRequest.ClientId,
			"client_secret": googleAuthRequest.ClientSecret,
			"grant_type":    googleAuthRequest.GrantType,
			"refresh_token": googleAuthRequest.RefreshToken,
		})
	httpResponse, err := json.Marshal(expectedGoogleAuthResponse)
	response := &http.Response{
		Body: httpResponse,
	}

	assert.NoError(t, err)

	httpClient := httpMocks.NewClient(t)
	httpClient.EXPECT().NewRequest().Return(http.NewRequest(nil))
	httpClient.EXPECT().Post(t.Context(), httpRequest).Return(response, nil)

	service := NewGoogleServiceWithInterfaces(httpClient)
	googleAuthResponse, err := service.GetAuthRefresh(t.Context(), googleAuthRequest)

	assert.NoError(t, err)
	assert.Equal(t, expectedGoogleAuthResponse.AccessToken, googleAuthResponse.AccessToken)
	assert.Equal(t, expectedGoogleAuthResponse.TokenType, googleAuthResponse.TokenType)
	assert.Equal(t, expectedGoogleAuthResponse.ExpiresIn, googleAuthResponse.ExpiresIn)
}

func TestGoogleService_GetAuthRefresh_Error(t *testing.T) {
	googleAuthRequest := &GoogleAuthRequest{
		ClientId:     "ClientId",
		ClientSecret: "ClientSecret",
		GrantType:    "GrantType",
		RefreshToken: "RefreshToken",
	}

	httpClient := httpMocks.NewClient(t)
	httpClient.EXPECT().NewRequest().Return(http.NewRequest(nil))
	httpClient.EXPECT().Post(t.Context(), mock.AnythingOfType("*http.Request")).Return(nil, errors.New("test"))

	service := NewGoogleServiceWithInterfaces(httpClient)
	_, err := service.GetAuthRefresh(t.Context(), googleAuthRequest)

	assert.Error(t, err)
}

func TestGoogleService_GetAuthRefresh_OAuthError(t *testing.T) {
	googleAuthRequest := &GoogleAuthRequest{
		ClientId:     "ClientId",
		ClientSecret: "ClientSecret",
		GrantType:    "GrantType",
		RefreshToken: "ExpiredRefreshToken",
	}
	errorResponse := &GoogleAuthResponse{
		Error:            "invalid_grant",
		ErrorDescription: "Token has been expired or revoked",
	}
	httpResponse, err := json.Marshal(errorResponse)
	assert.NoError(t, err)

	response := &http.Response{
		Body: httpResponse,
	}

	httpClient := httpMocks.NewClient(t)
	httpClient.EXPECT().NewRequest().Return(http.NewRequest(nil))
	httpClient.EXPECT().Post(t.Context(), mock.AnythingOfType("*http.Request")).Return(response, nil)

	service := NewGoogleServiceWithInterfaces(httpClient)
	_, err = service.GetAuthRefresh(t.Context(), googleAuthRequest)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid_grant")
	assert.ErrorContains(t, err, "Token has been expired or revoked")

	var authError *GoogleAuthError
	assert.ErrorAs(t, err, &authError)
	assert.Equal(t, "invalid_grant", authError.ErrorCode)
	assert.Equal(t, "Token has been expired or revoked", authError.ErrorDescription)
}
