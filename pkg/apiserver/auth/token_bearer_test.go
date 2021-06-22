package auth_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver/auth"
	"github.com/applike/gosoline/pkg/ddb"
	ddbMocks "github.com/applike/gosoline/pkg/ddb/mocks"
	kvStoreMocks "github.com/applike/gosoline/pkg/kvstore/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
)

type bearer struct {
	token string
}

func (b *bearer) GetToken() string {
	return b.token
}

type tokenBearerTestCase struct {
	bearerId  string
	token     string
	success   bool
	bearer    *bearer
	bearerErr error
}

type hasExpectations interface {
	AssertExpectations(t mock.TestingT) bool
}

type providerProvider func(test *tokenBearerTestCase) (auth.TokenBearerProvider, []hasExpectations)

func makeKvStoreProvider(test *tokenBearerTestCase) (auth.TokenBearerProvider, []hasExpectations) {
	repo := new(kvStoreMocks.KvStore)

	if test.bearerId != "" && test.token != "" {
		repo.On("Get", context.Background(), test.bearerId, &bearer{}).Run(func(args mock.Arguments) {
			m := args.Get(2).(*bearer)

			if test.bearer != nil {
				*m = *test.bearer
			}
		}).Return(test.bearer != nil, test.bearerErr).Once()
	}

	return auth.ProvideTokenBearerFromGetter(repo, func() auth.TokenBearer {
		return &bearer{}
	}), []hasExpectations{repo}
}

func makeDdbProvider(test *tokenBearerTestCase) (auth.TokenBearerProvider, []hasExpectations) {
	repo := new(ddbMocks.Repository)
	hasExpectation := []hasExpectations{repo}

	if test.bearerId != "" && test.token != "" {
		builder := new(ddbMocks.GetItemBuilder)
		builder.On("WithHash", test.bearerId).Return(builder).Once()
		hasExpectation = append(hasExpectation, builder)

		repo.On("GetItemBuilder").Return(builder).Once()

		repo.On("GetItem", context.Background(), builder, &bearer{}).Run(func(args mock.Arguments) {
			m := args.Get(2).(*bearer)

			if test.bearer != nil {
				*m = *test.bearer
			}
		}).Return(&ddb.GetItemResult{
			IsFound: test.bearer != nil,
		}, test.bearerErr).Once()
	}

	return auth.ProvideTokenBearerFromDdb(repo, func() auth.TokenBearer {
		return &bearer{}
	}), hasExpectation
}

func (test *tokenBearerTestCase) run(t *testing.T, providerProvider providerProvider) {
	logger := logMocks.NewLoggerMockedAll()
	provider, hasExpectations := providerProvider(test)

	headers := http.Header{}
	headers.Set("X-BEARER-ID", test.bearerId)
	headers.Set("X-BEARER-TOKEN", test.token)

	ginCtx := &gin.Context{
		Request: &http.Request{
			Header: headers,
		},
	}

	a := auth.NewTokenBearerAuthenticatorWithInterfaces(logger, "X-BEARER-ID", "X-BEARER-TOKEN", provider)
	valid, err := a.IsValid(ginCtx)

	assert.Equal(t, test.success, valid)

	if test.success {
		assert.NoError(t, err)
		subject := auth.GetSubject(ginCtx.Request.Context())
		assert.Equal(t, test.bearer, subject.Attributes[auth.AttributeTokenBearer])
	} else {
		assert.Error(t, err)
		assert.Equal(t, auth.InvalidTokenErr{}, err)
	}

	for _, hasExpectation := range hasExpectations {
		hasExpectation.AssertExpectations(t)
	}
}

func TestTokenBearerAuthenticator_IsValid(t *testing.T) {
	for name, test := range makeTokenBearerTestCases() {
		for providerName, provider := range map[string]providerProvider{
			"kvStore": makeKvStoreProvider,
			"ddb":     makeDdbProvider,
		} {
			t.Run(fmt.Sprintf("%s-%s", name, providerName), func(t *testing.T) {
				test.run(t, provider)
			})
		}
	}
}

func makeTokenBearerTestCases() map[string]tokenBearerTestCase {
	return map[string]tokenBearerTestCase{
		"noHeaders": {
			bearerId: "",
			token:    "",
			success:  false,
		},
		"no bearer": {
			bearerId:  "my bearer",
			token:     "my token",
			success:   false,
			bearer:    nil,
			bearerErr: nil,
		},
		"bearer error": {
			bearerId:  "my bearer",
			token:     "my token",
			success:   false,
			bearer:    nil,
			bearerErr: errors.New("this is an error"),
		},
		"invalid token": {
			bearerId: "my bearer",
			token:    "my token",
			success:  false,
			bearer: &bearer{
				token: "not my token",
			},
			bearerErr: nil,
		},
		"valid token": {
			bearerId: "my bearer",
			token:    "my token",
			success:  true,
			bearer: &bearer{
				token: "my token",
			},
			bearerErr: nil,
		},
	}
}
