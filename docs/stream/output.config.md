```yaml
stream:
  input:
    my-redis-input:
      type: redis
      family: example
      application: stream-redis-producer
      serverName: default
      key: my-example-stream
      batchSize: 10
```
 
##type
**type**: `string`, **default**: `null` **validate**: `null`

##family
**type**: `string`, **default**: `{family}` **validate**: `null`

##application
**type**: `string`, **default**: `{application}` **validate**: `null`

##serverName
**type**: `string`, **default**: `default` **validate**: `min=1`

##key
**type**: `string`, **default**: `null`, **validate**: `required,min=1`

##batchSize
**type**: `int`, **default**: `10`, **validate**: `required,min=1`