package env

import (
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestExternalServiceBindings(t *testing.T) {
	for _, typ := range []string{componentMySql, componentRedis, componentMailpit, componentWiremock, componentDynamoDb, componentS3} {
		t.Run(typ, func(t *testing.T) {
			config := cfg.New()
			require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
				"test.container_manager.runner_type":  "external",
				"test.components." + typ + ".default": map[string]any{"host": "service", "port": 18000, "web_port": 18001},
			})))
			factory := componentFactories[typ]
			settings := factory.GetSettingsSchema()
			require.NoError(t, UnmarshalSettings(config, settings, typ, "default"))
			desc := factory.DescribeContainers(settings)["main"]
			require.Equal(t, RunnerTypeExternal, desc.ContainerConfig.RunnerType)
			runner, err := NewContainerRunnerExternal(config, log.NewLogger(), nil)
			require.NoError(t, err)
			container, err := runner.RunContainer(t.Context(), ContainerRequest{ContainerDescription: desc})
			require.NoError(t, err)
			require.Equal(t, "service:18000", container.bindings["main"].getAddress())
			if typ == componentMailpit {
				require.Equal(t, "service:18001", container.bindings["web"].getAddress())
			}
		})
	}
}

func TestDynamoDbNamingIsolation(t *testing.T) {
	factory := new(dynamoDbFactory)
	containers := map[string]*Container{"main": {bindings: map[string]ContainerBinding{"main": {host: "localhost", port: "8000"}}}}
	patterns := make([]string, 0, 2)
	for range 2 {
		config := cfg.New()
		require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
			"cloud.aws.s3.clients.default.endpoint":                   "http://s3:5000",
			"cloud.aws.dynamodb.clients.default.naming.table_pattern": "{app.env}-{name}",
		})))
		component, err := factory.Component(config, nil, containers, &dynamoDbSettings{Region: "eu-central-1"})
		require.NoError(t, err)
		require.NoError(t, config.Option(component.(ComponentCfgOptionAware).CfgOptions()...))
		naming, err := ddb.GetTableNamingSettings(config, "default")
		require.NoError(t, err)
		require.Contains(t, naming.TablePattern, "-{app.env}-{name}")
		patterns = append(patterns, naming.TablePattern)
		s3Endpoint, err := config.GetString("cloud.aws.s3.clients.default.endpoint")
		require.NoError(t, err)
		require.Equal(t, "http://s3:5000", s3Endpoint)
	}
	require.NotEqual(t, patterns[0], patterns[1])
}

func TestS3BucketIsolation(t *testing.T) {
	factory := new(s3Factory)
	containers := map[string]*Container{"main": {bindings: map[string]ContainerBinding{"main": {host: "localhost", port: "5000"}}}}
	var buckets []string
	for range 2 {
		config := cfg.New()
		require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
			"app.env": "test", "app.name": "blob-test", "app.namespace": "gosoline-test",
			"blob.first.bucket": "shared-bucket", "blob.first.prefix": "original-prefix",
			"blob.second.bucket":                          "shared-bucket",
			"blob.longfirst.bucket":                       strings.Repeat("a", 62) + "b",
			"blob.longsecond.bucket":                      strings.Repeat("a", 62) + "c",
			"cloud.aws.dynamodb.clients.default.endpoint": "http://dynamodb:8000",
		})))
		component, err := factory.Component(config, nil, containers, &s3Settings{Region: "eu-central-1"})
		require.NoError(t, err)
		require.NoError(t, config.Option(component.(ComponentCfgOptionAware).CfgOptions()...))
		bucket, err := config.GetString("blob.first.bucket")
		require.NoError(t, err)
		otherBucket, err := config.GetString("blob.second.bucket")
		require.NoError(t, err)
		require.Equal(t, bucket, otherBucket)
		longFirst, err := config.GetString("blob.longfirst.bucket")
		require.NoError(t, err)
		longSecond, err := config.GetString("blob.longsecond.bucket")
		require.NoError(t, err)
		require.Len(t, longFirst, 63)
		require.NotEqual(t, longFirst, longSecond)
		buckets = append(buckets, bucket)
		prefix, err := config.GetString("blob.first.prefix")
		require.NoError(t, err)
		require.Equal(t, "original-prefix", prefix)
		ddbEndpoint, err := config.GetString("cloud.aws.dynamodb.clients.default.endpoint")
		require.NoError(t, err)
		require.Equal(t, "http://dynamodb:8000", ddbEndpoint)
	}
	require.NotEqual(t, buckets[0], buckets[1])
}

func TestExternalMySqlDatabaseIsolation(t *testing.T) {
	config := cfg.New()
	require.NoError(t, config.Option(cfg.WithConfigSetting("test.container_manager.runner_type", "external")))
	factory := new(mysqlFactory)
	var databases []string
	for range 2 {
		settings := new(mysqlSettings)
		require.NoError(t, UnmarshalSettings(config, settings, componentMySql, "default"))
		desc := factory.DescribeContainers(settings)["main"]
		require.Equal(t, 3306, desc.ContainerConfig.PortBindings["main"].ContainerPort)
		databases = append(databases, settings.Credentials.DatabaseName)
	}
	require.NotEqual(t, databases[0], databases[1])
}

func TestRedisDatabaseIsolation(t *testing.T) {
	server, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(server.Close)
	container := &Container{bindings: map[string]ContainerBinding{
		"main": {host: server.Host(), port: server.Port()},
	}}
	factory := new(redisFactory)
	first := factory.client(container, 1)
	second := factory.client(container, 2)
	t.Cleanup(func() {
		require.NoError(t, first.Close())
		require.NoError(t, second.Close())
	})
	ctx := context.Background()
	require.NoError(t, first.Set(ctx, "same-key", "first", 0).Err())
	require.NoError(t, second.Set(ctx, "same-key", "second", 0).Err())
	require.NoError(t, first.FlushDB(ctx).Err())
	value, err := second.Get(ctx, "same-key").Result()
	require.NoError(t, err)
	require.Equal(t, "second", value)
	component, err := factory.Component(nil, nil, map[string]*Container{"main": container}, &redisSettings{DB: 2})
	require.NoError(t, err)
	config := cfg.New()
	require.NoError(t, config.Option(component.(ComponentCfgOptionAware).CfgOptions()...))
	db, err := config.GetInt("redis.default.db")
	require.NoError(t, err)
	require.Equal(t, 2, db)
}
