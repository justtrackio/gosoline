package grpcclient

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcClientCtxKey string

const (
	grpcSecurityInsecure = "insecure"
)

type GlobalSettings struct {
	Domain string `cfg:"domain" default:""`
}

type Settings struct {
	Domain   string `cfg:"domain" default:""`
	Endpoint string `cfg:"endpoint" default:""`
	Family   string `cfg:"family" default:""`
	Port     int    `cfg:"port" default:"8081"`

	Security string `cfg:"security" default:"insecure"`
}

type ServiceFactory[T any] func(connInterface grpc.ClientConnInterface) T

func defaultServiceNamingStrategy(appId cfg.AppId, settings *Settings, service string) string {
	return fmt.Sprintf("%s.%s.%s.%s", service, settings.Family, appId.Environment, settings.Domain)
}

func ProvideClient[T any](ctx context.Context, config cfg.Config, service string, buildService ServiceFactory[T]) (*T, error) {
	appId := &cfg.AppId{}
	appId.PadFromConfig(config)

	settings := &Settings{}
	config.UnmarshalKey(fmt.Sprintf("grpc_client.%s", service), settings)

	global := &GlobalSettings{}
	config.UnmarshalKey("grpc_client", global)

	conn, err := appctx.Provide(ctx, grpcClientCtxKey(fmt.Sprintf("conn_%s", service)), func() (*grpc.ClientConn, error) {
		if settings.Endpoint == "" {
			settings.Endpoint = defaultServiceNamingStrategy(*appId, settings, service)
		}

		if settings.Domain == "" {
			settings.Domain = global.Domain
		}

		endpoint := fmt.Sprintf("%s:%d", settings.Endpoint, settings.Port)

		var opts []grpc.DialOption
		if settings.Security == grpcSecurityInsecure {
			opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}

		return grpc.Dial(endpoint, opts...)
	})
	if err != nil {
		return nil, err
	}

	s := buildService(conn)
	return &s, nil
}
