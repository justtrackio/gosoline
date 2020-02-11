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

db_drivername: "mysql"
db_hostname: "127.0.0.1"
db_port: 3306
db_database: "examples"
db_username: "root"
db_password: "mcoins"
db_retry_wait: 10
db_health_check_delay: 30
db_max_connection_lifetime: 120
db_parse_time: true
db_auto_migrate: true
db_migrations_path: file://../../build/migrations/mysql-crud
db_table_prefixed: true

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

redis_kvstore_currency_mode: "discover"
redis_kvstore_currency_addr: ""

stream:
  consumer:
    default:
      input: consumer
      runner_count: 10
      idle_timeout: 5s
      encoding: application/json

  producer:
    default:
      encoding: application/json
      output: sqs-out

  input:
    consumer:
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

subscriptions:
  - input: sns
    output: kvstore
    source: { family: example, application: mysql-crud, name: yourModel }

tracing:
  provider: xray
  enabled: true
  addr_type: local
  addr_value: ""
```