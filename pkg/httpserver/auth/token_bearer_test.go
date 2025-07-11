package auth_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/ddb"
	ddbMocks "github.com/justtrackio/gosoline/pkg/ddb/mocks"
	"github.com/justtrackio/gosoline/pkg/httpserver/auth"
	kvStoreMocks "github.com/justtrackio/gosoline/pkg/kvstore/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
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

type providerProvider func(test *tokenBearerTestCase) auth.TokenBearerProvider

func makeKvStoreProvider(t *testing.T, test *tokenBearerTestCase) auth.TokenBearerProvider {
	repo := kvStoreMocks.NewKvStore[bearer](t)

	if test.bearerId != "" && test.token != "" {
		repo.EXPECT().Get(matcher.Context, test.bearerId, &bearer{}).Run(func(ctx context.Context, key any, m *bearer) {
			if test.bearer != nil {
				*m = *test.bearer
			}
		}).Return(test.bearer != nil, test.bearerErr).Once()
	}

	return auth.ProvideTokenBearerFromGetter(func(ctx context.Context, key string, value auth.TokenBearer) (bool, error) {
		return repo.Get(ctx, key, value.(*bearer))
	}, func() auth.TokenBearer {
		return &bearer{}
	})
}

func makeDdbProvider(t *testing.T, test *tokenBearerTestCase) auth.TokenBearerProvider {
	repo := ddbMocks.NewRepository(t)

	if test.bearerId != "" && test.token != "" {
		builder := ddbMocks.NewGetItemBuilder(t)
		builder.EXPECT().WithHash(test.bearerId).Return(builder).Once()

		repo.EXPECT().GetItemBuilder().Return(builder).Once()

		repo.EXPECT().GetItem(matcher.Context, builder, &bearer{}).Run(func(ctx context.Context, qb ddb.GetItemBuilder, result any) {
			m := result.(*bearer)

			if test.bearer != nil {
				*m = *test.bearer
			}
		}).Return(&ddb.GetItemResult{
			IsFound: test.bearer != nil,
		}, test.bearerErr).Once()
	}

	return auth.ProvideTokenBearerFromDdb(repo, func() auth.TokenBearer {
		return &bearer{}
	})
}

func (test *tokenBearerTestCase) run(t *testing.T, providerProvider providerProvider) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	provider := providerProvider(test)

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
}

func TestTokenBearerAuthenticator_IsValid(t *testing.T) {
	for name, test := range makeTokenBearerTestCases() {
		for providerName, provider := range map[string]providerProvider{
			"kvStore": func(test *tokenBearerTestCase) auth.TokenBearerProvider {
				return makeKvStoreProvider(t, test)
			},
			"ddb": func(test *tokenBearerTestCase) auth.TokenBearerProvider {
				return makeDdbProvider(t, test)
			},
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
