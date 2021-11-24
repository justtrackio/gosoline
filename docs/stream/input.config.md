#Redis
```yaml
stream:
  input:
    my-redis-input:
      type: redis
      family: example
      application: stream-redis-producer
      server_name: default
      key: my-example-stream
      wait_time: 3s
```

```go:code_snippet.go
```

##type
**type**: `string`, **default**: `null` **validate**: `required`

##family
**type**: `string`, **default**: `{family}` **validate**: `null`

##application
**type**: `string`, **default**: `{application}` **validate**: `null`

##server_name
**type**: `string`, **default**: `default` **validate**: `min=1`

##key
**type**: `string`, **default**: `null`, **validate**: `required,min=1`

##wait_time
**type**: `time.Duration`, **default**: `3s`

#SNS
```yaml
stream:
  input:
    my-sns-input:
      type: sns
      wait_time: 3
      visibility_timeout: sns-producer
      redrive_policy:
        enabled: true
        max_receive_count: 3
      targets:
        - { family: example, application: stream-sns-producer, topic_id: foobar }
      runner_count: 1
```
 
##type
**type**: `string`, **default**: `null` **validate**: `required`

##wait_time
**type**: `int`, **default**: `3` **validate**: `min=1`

##visibility_timeout
**type**: `int`, **default**: `30` **validate**: `min=1`

##redrive_policy.enabled
**type**: `bool`, **default**: `true` **validate**: `null`

##redrive_policy.max_receive_count
**type**: `int`, **default**: `3` **validate**: `null`

##targets
**type**: `map`, **default**: `null` **validate**: `min=1`

##runner_count
**type**: `int`, **default**: `1` **validate**: `min=1`

#SNS
```yaml
stream:
  input:
    consumer:
      type: sqs
      target_family: example
      target_application: stream-sqs-producer
      target_queue_id: foobar
      wait_time: 1
      visibility_timeout: 10
      fifo:
        enabled: false
        contentBasedDeduplication: false
      redrive_policy:
        enabled: true
        max_receive_count: 3
      runner_count: 2
```
 
##type
**type**: `string`, **default**: `null` **validate**: `required`

##target_family
**type**: `string`, **default**: `{family}` **validate**: `min=1`

##target_application
**type**: `string`, **default**: `{family}` **validate**: `min=1`

##target_queue_id
**type**: `string`, **default**: `{family}` **validate**: `min=1`

##wait_time
**type**: `int`, **default**: `3` **validate**: `min=1`

##visibility_timeout
**type**: `int`, **default**: `30` **validate**: `min=1`

##fifo.enabled
**type**: `bool`, **default**: `false` **validate**: `null`

##fifo.content_based_deduplication
**type**: `bool`, **default**: `false` **validate**: `null`

##redrive_policy.enabled
**type**: `bool`, **default**: `true` **validate**: `null`

##redrive_policy.max_receive_count
**type**: `int`, **default**: `3` **validate**: `null`

##runner_count
**type**: `int`, **default**: `1` **validate**: `min=1`
