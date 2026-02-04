# Migration Guide: `app-tags` and Resource Naming

This guide details how to migrate your gosoline application to the new `AppIdentity` system and dynamic resource naming introduced in the `app-tags` branch.

## 1. Update Configuration (`config.yml` / `.env`)

The flat application hierarchy (`project`, `family`, `group`) has been replaced with a flexible tagging system. You must move these keys into the `app.tags` structure.

### Basic Identity

**Old `config.yml`:**
```yaml
env: production
app_project: justtrack
app_family: platform
app_group: core
app_name: my-service
```

**New `config.yml`:**
```yaml
app:
  env: production
  name: my-service
  tags:
    project: justtrack
    family: platform
    group: core
```

### Environment Variables

If you configure your app via environment variables, update the keys:

| Old Variable | New Variable |
| :--- | :--- |
| `GOSO_APP_PROJECT` | `GOSO_APP_TAGS_PROJECT` |
| `GOSO_APP_FAMILY` | `GOSO_APP_TAGS_FAMILY` |
| `GOSO_APP_GROUP` | `GOSO_APP_TAGS_GROUP` |
| `GOSO_APP_NAME` | `GOSO_APP_NAME` (no change) |
| `GOSO_ENV` | `GOSO_APP_ENV` |

## 2. Update Resource Naming Patterns

If you have customized the naming patterns for AWS resources (SQS, SNS, Kinesis, DynamoDB) or Kafka in your config, you must update the placeholders.

### Placeholder Mapping

| Old Placeholder | New Placeholder | Notes |
| :--- | :--- | :--- |
| `{project}` | `{app.tags.project}` | Requires `project` tag |
| `{family}` | `{app.tags.family}` | Requires `family` tag |
| `{group}` | `{app.tags.group}` | Requires `group` tag |
| `{env}` | `{app.env}` | Built-in field |
| `{app}` | `{app.name}` | Built-in field |
| `{modelId}` | **REMOVED** | See DynamoDB section below |

### Example: SQS Pattern Update

**Old:**
```yaml
cloud:
  aws:
    sqs:
      clients:
        default:
          naming:
            pattern: "{project}-{env}-{queueId}"
```

**New:**
```yaml
cloud:
  aws:
    sqs:
      clients:
        default:
          naming:
            pattern: "{app.tags.project}-{app.env}-{queueId}"
```

### Special Case: DynamoDB Table Naming

For DynamoDB tables, the naming logic has been aligned with other AWS resources.
*   **Placeholder Change:** Use `{name}` instead of `{modelId}` to refer to the model name.
*   **No Auto-Append:** Unlike the canonical Model ID, the table name pattern **does not** automatically append the model name. You must explicitly include `{name}` in your pattern if you want it (which you almost certainly do).

*   **Old:** `{project}-{env}-{modelId}` (or `{modelId}` was implied/appended in some contexts)
*   **New:** `{app.tags.project}-{app.env}-{name}`

**Migration:**
1.  Replace `{modelId}` with `{name}`.
2.  Ensure `{name}` is present in your pattern.

## 3. Update Go Code

### `cfg.AppId` is now `cfg.AppIdentity`

The `AppId` struct and its getters have been removed.

**Migration:**

```go
// OLD
appId, _ := cfg.GetAppIdFromConfig(config)
fmt.Println(appId.Project)

// NEW
identity, _ := cfg.GetAppIdentityFromConfig(config)
fmt.Println(identity.Tags["project"])
```

### `mdl.ModelId` Refactoring

The `mdl.ModelId` struct no longer has explicit hierarchy fields.

**Migration:**

```go
// OLD
modelId := mdl.ModelId{
    Project: "my-project",
    Family:  "my-family",
    Name:    "my-model",
}

// NEW
modelId := mdl.ModelId{
    Name: "my-model",
    Tags: map[string]string{
        "project": "my-project",
        "family":  "my-family",
    },
}
```

### ModelId Domain Pattern (Canonical String Representation)

The string representation of a `ModelId` (used for message routing attributes, etc.) is now configurable via `app.model_id.domain_pattern`.

*   **Config Key:** `app.model_id.domain_pattern`
*   **Old Default:** (Implicitly hardcoded as `{project}.{env}.{family}.{group}`)
*   **New Default:** There is no hardcoded default; you should configure this if you rely on `modelId.String()`.
*   **Recommended Value:** `{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}`

If this pattern is missing, calling `modelId.String()` will return an error.

**Important:**
1.  The `{modelId}` placeholder is **NOT** used in this pattern. The model name is automatically appended as the last segment.
2.  **Strict Delimiter:** The pattern must consist *only* of placeholders separated by **single dots (`.`)**. No other characters or delimiters (like dashes) are allowed between placeholders. This ensures the ID can be reliably parsed back.

*   **Valid:** `{app.tags.project}.{app.env}`
*   **Invalid:** `{app.tags.project}-{app.env}` (dashes not allowed)
*   **Invalid:** `prefix.{app.tags.project}` (static text not allowed)

*   Pattern: `{app.tags.project}.{app.env}`
*   Model Name: `myModel`
*   Result: `myProject.production.myModel`

### `pkg/parquet` Removed

The `pkg/parquet` package has been deleted. Remove any imports or usage of this package from your codebase.

## 4. Redis Key Prefixing (Optional)

A new feature allows automatic key prefixing for Redis. If you were manually prefixing keys in your application code, you can now move this to configuration:

```yaml
redis:
  clients:
    default:
      naming:
        key_pattern: "{app.tags.project}:{app.name}:{key}"
```

## 5. Reference: Naming Patterns & Environment Variables

This section lists the configuration keys, their old and new defaults, and the corresponding environment variables for overriding them.

### SQS Queue Naming

*   **Config Key:** `cloud.aws.sqs.clients.default.naming.pattern`
*   **Old Default:** `{project}-{env}-{family}-{group}-{queueId}`
*   **New Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{queueId}`
*   **Environment Variable:** `GOSO_CLOUD_AWS_SQS_CLIENTS_DEFAULT_NAMING_PATTERN`

### SNS Topic Naming

*   **Config Key:** `cloud.aws.sns.clients.default.naming.pattern`
*   **Old Default:** `{project}-{env}-{family}-{group}-{topicId}`
*   **New Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{topicId}`
*   **Environment Variable:** `GOSO_CLOUD_AWS_SNS_CLIENTS_DEFAULT_NAMING_PATTERN`

### Kinesis Naming

**Stream Pattern:**
*   **Config Key:** `cloud.aws.kinesis.clients.default.naming.stream_pattern` (Renamed from `pattern`)
*   **Old Default:** `{project}-{env}-{family}-{group}-{streamName}`
*   **New Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{streamName}`
*   **Environment Variable:** `GOSO_CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_STREAM_PATTERN`

**Metadata Table Pattern:**
*   **Config Key:** `cloud.aws.kinesis.clients.default.naming.metadata_pattern` (New)
*   **New Default:** `{app.env}-kinsumer-metadata`
*   **Environment Variable:** `GOSO_CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_METADATA_PATTERN`

### Kafka Naming

**Topic Pattern:**
*   **Config Key:** `kafka.naming.topic_pattern`
*   **Old Default:** `{project}-{env}-{family}-{group}-{topicId}`
*   **New Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{topicId}`
*   **Environment Variable:** `GOSO_KAFKA_NAMING_TOPIC_PATTERN`

**Consumer Group Pattern:**
*   **Config Key:** `kafka.naming.group_pattern`
*   **Old Default:** `{project}-{env}-{family}-{group}-{app}-{groupId}`
*   **New Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{app.name}-{groupId}`
*   **Environment Variable:** `GOSO_KAFKA_NAMING_GROUP_PATTERN`

### DynamoDB Table Naming

*   **Config Key:** `cloud.aws.dynamodb.clients.default.naming.pattern`
*   **Old Default:** `{project}-{env}-{family}-{group}-{modelId}`
*   **New Default:** `{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{name}`
*   **Environment Variable:** `GOSO_CLOUD_AWS_DYNAMODB_CLIENTS_DEFAULT_NAMING_PATTERN`

### CloudWatch Namespace

*   **Config Key:** `metric.writer.cloudwatch.naming.pattern`
*   **Old Default:** `{project}/{env}/{family}/{group}-{app}`
*   **New Default:** `{app.tags.project}/{app.env}/{app.tags.family}/{app.tags.group}-{app.name}`
*   **Environment Variable:** `GOSO_METRIC_WRITER_CLOUDWATCH_NAMING_PATTERN`

### Redis Key Prefix
*   **Config Key:** `redis.clients.default.naming.key_pattern`
*   **Old Default:** N/A (Feature did not exist)
*   **New Default:** `{key}` (No prefix)
*   **Environment Variable:** `GOSO_REDIS_CLIENTS_DEFAULT_NAMING_KEY_PATTERN`

### Blob / S3 Bucket Patterns
The `blob` package now delegates bucket naming to the `cloud.aws.s3` package. The configuration keys `blob.<store_name>.bucket_pattern` and `blob.default.bucket_pattern` have been **removed**.

*   **Config Key:** `cloud.aws.s3.clients.<client_name>.naming.bucket_pattern`
*   **Old Default:** `{app.tags.project}-{app.env}-{app.tags.family}`
*   **New Default:** `{app.tags.project}-{app.env}-{app.tags.family}` (Unchanged)
*   **New Placeholder:** `{bucketId}` is now available (resolves to the blob store name).

**Migration:**
Move any custom bucket patterns to the new S3 naming config key. If you used `blob.default.bucket_pattern`, move it to `cloud.aws.s3.clients.default.naming.bucket_pattern`.

**Example Env Var Usage:**

```bash
# Override SQS naming to just use env and queueId
export GOSO_CLOUD_AWS_SQS_CLIENTS_DEFAULT_NAMING_PATTERN="{app.env}-{queueId}"

# Set a redis key prefix
export GOSO_REDIS_CLIENTS_DEFAULT_NAMING_KEY_PATTERN="{app.name}:{key}"
```

## 6. Stream Input/Output Configuration

The configuration for stream inputs and outputs has been updated to use the new `AppIdentity` structure.

### General Change
For most inputs (Redis) and outputs (Kafka, Kinesis, Redis, SNS, SQS), the flat fields `project`, `family`, `group`, and `application` have been replaced by an `identity` block.

**Old Output Config:**
```yaml
stream:
  output:
    my-output:
      type: sqs
      project: my-project
      family: my-family
      group: my-group
      queue_id: my-queue
```

**New Output Config:**
```yaml
stream:
  output:
    my-output:
      type: sqs
      identity:
        tags:
          project: my-project
          family: my-family
          group: my-group
      queue_id: my-queue
```

### SQS Input
For SQS inputs, the `target_*` fields for identity have been grouped into `target_identity`.

**Old:**
```yaml
stream:
  input:
    my-input:
      type: sqs
      target_family: my-family
      target_group: my-group
      target_queue_id: my-queue
```

**New:**
```yaml
stream:
  input:
    my-input:
      type: sqs
      target_identity:
        tags:
          family: my-family
          group: my-group
      target_queue_id: my-queue
```

### SNS Input
For SNS inputs, the consumer identity and the target topic identities have been updated.

**Old:**
```yaml
stream:
  input:
    my-sns-input:
      type: sns
      id: my-consumer
      family: my-family
      group: my-group
      targets:
        - family: target-family
          group: target-group
          topic_id: target-topic
```

**New:**
```yaml
stream:
  input:
    my-sns-input:
      type: sns
      id: my-consumer
      identity:
        tags:
          family: my-family
          group: my-group
      targets:
        - identity:
            tags:
              family: target-family
              group: target-group
          topic_id: target-topic
```
