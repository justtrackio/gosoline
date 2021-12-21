## Configuration
The AWS SDK v2 based services use the following default settings for region 
and endpoint, meaning you get those values for every requested client if you 
don't specify anything else for the client.
```json
cloud:
  aws:
    defaults:
      region: "eu-central-1"
      endpoint: "http://localhost:4566" #localstack
```

### General service config
AWS service clients are created and configured by name. The pattern here is:
```golang
func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...func(options *awsCfg.LoadOptions) error) (*serviceX.Client, error) 
```
The resulting client can be configured by:
```yaml
cloud:
    aws:
        serviceX:
            clients:
                default: # name of the client
                    endpoint: "http://localhost:4566"
                    region: "eu-central-1"
                    http_client:
                        timeout: 0s
                    backoff:
                        cancel_delay: 1s
                        initial_interval: 50ms
                        max_attempts: 10
                        max_elapsed_time: 15m0s
                        max_interval: 10s 
```
| Setting                  | Description                                                                        | Default                       |
|--------------------------|------------------------------------------------------------------------------------|-------------------------------|
| endpoint                 | Which service endpoint should be called                                            | http://localhost:4566         |
| region                   | The region in use                                                                  | eu-central-1                  |
| http_client.timeout      | After what duration the request should be canceled, if the server doesn't respond | 0s (no timeout)               |
| backoff.cancel_delay     | If the request get canceled, how long should the cancel delayed                    | 1s                            |
| backoff.initial_interval | The initial duration to wait before retrying the request on error                  | 50ms                          |
| backoff.max_attempts     | How many attempts should be done                                                   | 10 (0 means retry forever)    |
| backoff.max_interval     | Max duration between 2 calls when retrying                                         | 10s                           |
| backoff.max_elapsed_time | For how long the service should retry the request                                  | 10m (0m means retry forever)  |

### Cloudwatch
Call 
```golang
import(
    gosoCloudwatch "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch"
)

cloudwatchClient := gosoCloudwatch.ProvideClient(ctx, config, logger , "default")
```
to get the default cloudwatch client. You don't have to provide any config 
if you want to go with the default settings. The default settings are:
```yaml
cloud:
    aws:
        cloudwatch:
            clients:
                default:
                    endpoint: "http://localhost:4566"
                    region: "eu-central-1"
                    http_client:
                      timeout: 0s
                    backoff:
                      cancel_delay: 1s
                      initial_interval: 50ms
                      max_attempts: 10
                      max_elapsed_time: 15m0s
                      max_interval: 10s 
```
These are the default values which are used if you don't provide any config by yourself. 
