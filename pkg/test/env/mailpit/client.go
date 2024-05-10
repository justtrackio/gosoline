package mailpit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	urlPkg "net/url"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate go run github.com/vektra/mockery/v2 --name Client
type Client interface {
	ListMessages(ctx context.Context) (*ListMessagesResponse, error)
	GetMessage(ctx context.Context, id string) (*Message, error)
}

type client struct {
	client http.Client
	config Config
}

type Config struct {
	Server   string `cfg:"server"`
	Protocol string `cfg:"protocol" default:"http"`
}

type mailpitCtxKey string

func ProvideClient(ctx context.Context, config cfg.Config, _ log.Logger) (Client, error) {
	return appctx.Provide(ctx, mailpitCtxKey("default"), func() (Client, error) {
		var conf Config
		if err := config.UnmarshalKey("mailpit", &conf); err != nil {
			return nil, fmt.Errorf("failed to unmarshal mailpit config: %w", err)
		}

		if conf.Server == "" {
			return nil, fmt.Errorf("mailpit server is required")
		}

		return NewClientWithInterfaces(http.Client{}, conf), nil
	})
}

func NewClientWithInterfaces(httpClient http.Client, conf Config) *client {
	return &client{
		client: httpClient,
		config: conf,
	}
}

func (c client) ListMessages(_ context.Context) (*ListMessagesResponse, error) {
	resp, err := c.client.Get(c.endpoint("/api/v1/messages"))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	var listResponse ListMessagesResponse
	if err := json.Unmarshal(body, &listResponse); err != nil {
		return nil, err
	}

	return &listResponse, nil
}

func (c client) GetMessage(_ context.Context, id string) (*Message, error) {
	path := fmt.Sprintf("/api/v1/message/%s", id)
	req, err := http.NewRequest(http.MethodGet, c.endpoint(path), http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	var messageResponse Message
	if err := json.Unmarshal(body, &messageResponse); err != nil {
		return nil, err
	}

	return &messageResponse, nil
}

func (c client) endpoint(path string) string {
	url := urlPkg.URL{}
	url.Scheme = c.config.Protocol
	url.Host = c.config.Server
	url.Path = path

	return url.String()
}
