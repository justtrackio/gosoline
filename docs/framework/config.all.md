```yml
env: dev

app_project: mcoins
app_family: example
app_name: stream-sqs-consumer

api:
  health:
    port: 0

api_port: 8090
api_mode: release
api_timeout_read: 60
api_timeout_write: 60
api_timeout_idle: 60

aws_sdk_retries: 1
aws_cloudwatch_endpoint: http://localhost:4582
aws_dynamoDb_endpoint: http://localhost:4569
aws_dynamoDb_autoCreate: false
aws_sns_endpoint: http://localhost:4575
aws_sns_autoSubscribe: false
aws_sqs_endpoint: http://localhost:4576
aws_sqs_autoCreate: false

db:
  default:
    driver: mysql
    max_connection_lifetime: 120
    parse_time: true
    uri:
      host: 127.0.0.1
      port: 3307
      user: root
      password: mcoins
      database: examples
    migrations:
      enabled: true
      table_prefixed: true
      path: file://../../build/migrations/mysql-crud

kvstore:
  currency:
    type: chain
    elements: [redis, ddb]

mon:
  logger:
    level: info
    format: console
    timestamp_format: 15:04:05.000
    tags: {}
  metric:
    enabled: false
    writers: [cw]
    interval: 60s

redis_default_currency_mode: "discover"
redis_default_currency_addr: ""
redis_kvstore_currency_mode: "discover"
redis_kvstore_currency_addr: ""

stream:
  consumer:
    default:
      input: consumer-sqs
      runner_count: 10
      idle_timeout: 5s
      encoding: application/json

  producer:
    default:
      encoding: application/json
      compression: application/gzip
      output: sqs-out

  input:
    consumer-redis:
      type: redis      
      family: example
      application: stream-redis-producer
      server_name: default
      key: my-example-stream
      wait_time: 1s

    consumer-sqs:
      type: sqs
      target_queue_id: postbackTypeEvent
      wait_time: 5
      visibility_timeout: 1800
      runner_count: 10
      fifo:
        enabled: true
        content_based_deduplication: true
      redrive_policy:
        enabled: true
        max_receive_count: 4
      backoff:
        enabled: true
        blocking: true
        cancel_delay: 6s

  output:
    redis:
      type: redis
      project: gosoline
      family: example
      application: redis-producer
      server_name: default
      key: my-prefix
      batch_size: 10

    sns:
      type: sns
      project: gosoline
      family: example
      application: redis-producer
      topic_id: foobar
      client:
        max_retries: 1
        http_timeout: 3s
        log_level: debug_request_errors
      backoff:
        enabled: true
        blocking: false
        cancel_delay: 1s
        initial_interval: 500ms
        randomization_factor: 0.5
        multiplier: 1.5
        max_interval: 3s
        max_elapsed_time: 15m

subscriptions:
  - input: sns
    output: kvstore
    source: { family: example, application: mysql-crud, name: yourModel }

tracing:
  provider: xray
  enabled: true
  addr_type: local
  addr_value: ""
  sampling:
      version: 1
      default:
        description: default
        fixed_target: 1
        rate: 0.05
      rules:
        - { description: sample-service, service_name: "{app_project}-{env}-{app_family}-{app_name}", http_method: "*", url_path: "*", fixed_target: 0, rate: 0.05}

test:
  logger:
    level: info
    format: console
    timestamp_format: 15:04:05.000

  container_runner:
    endpoint: ""
    name_prefix: "goso"
    health_check:
      initial_interval: 1s
      max_interval: 3s
      max_elapsed_time: 1m

  components:
      - type: streamInput
        name: consumer
  
      - type: mysql
        name: default
        expire_after: 2m
        version: 8
        port: 0
        credentials:
          database_name: gosoline
          user_name: gosoline
          user_password: gosoline
          root_password: gosoline
```
