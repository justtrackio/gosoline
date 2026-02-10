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
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
  tags:
    project: justtrack
    family: platform
    group: core
```

**Note:** The `app.namespace` configuration is optional but recommended. It allows you to define a reusable namespace pattern that can be referenced as `{app.namespace}` in all resource naming patterns. See the "Namespace Pattern" section below for details.

### Environment Variables

If you configure your app via environment variables, update the keys:

| Old Variable | New Variable |
| :--- | :--- |
| `GOSO_APP_PROJECT` | `GOSO_APP_TAGS_PROJECT` |
| `GOSO_APP_FAMILY` | `GOSO_APP_TAGS_FAMILY` |
| `GOSO_APP_GROUP` | `GOSO_APP_TAGS_GROUP` |
| `GOSO_APP_NAME` | `GOSO_APP_NAME` (no change) |
| `GOSO_ENV` | `GOSO_APP_ENV` |
| N/A (new feature) | `GOSO_APP_NAMESPACE` |

## 2. Update Resource Naming Patterns

**IMPORTANT:** The default naming patterns for most services have changed to use `{app.namespace}` instead of explicit tag placeholders. If you don't configure `app.namespace`, your resource names will be missing the project/family/group prefix. For backward compatibility, you should configure:

```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

This ensures your resource names remain consistent with the previous defaults.

If you have customized the naming patterns for AWS resources (SQS, SNS, Kinesis, DynamoDB), Kafka, or Tracing in your config, you must update the placeholders.

### Namespace Pattern (New Feature)

A new `app.namespace` configuration option allows you to define a reusable namespace pattern that can be referenced in all resource naming patterns.

**Config Key:** `app.namespace`
**Format:** A pattern using dots (`.`) as delimiters, with standard `{app.*}` and `{app.tags.*}` placeholders
**Usage:** Reference as `{app.namespace}` in any resource naming pattern

**Example:**
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
  tags:
    project: justtrack
    family: platform
    group: core
  env: production

cloud:
  aws:
    sqs:
      clients:
        default:
          naming:
            # {app.namespace} expands to: justtrack-production-platform-core
            queue_pattern: "{app.namespace}-{queueId}"
```

**Benefits:**
- Define your naming hierarchy once and reuse it everywhere
- Simplify resource patterns by using `{app.namespace}` instead of repeating the same placeholders
- Easy to change your naming convention across all resources in one place

**Note:** When expanded, the dots in the namespace pattern are replaced with the service-specific delimiter (usually `-`). If no namespace is configured, `{app.namespace}` expands to an empty string.

### Delimiter Configuration

Each resource naming pattern has a paired **delimiter** configuration that controls how `{app.namespace}` is expanded:

- **Most services** use `identity.Format()` with a configurable delimiter (default is usually `-`)
- The delimiter replaces dots in the `{app.namespace}` pattern during expansion
- Example: namespace `{app.tags.project}.{app.env}` with delimiter `-` becomes `myproject-prod`
- Each service's delimiter can be customized via its `naming.delimiter` configuration key

All services now use `identity.Format()` with configurable delimiters.

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
            queue_pattern: "{app.tags.project}-{app.env}-{queueId}"
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

*   **Config Key:** `cloud.aws.sqs.clients.default.naming.queue_pattern`
*   **Old Default:** `{project}-{env}-{family}-{group}-{queueId}`
*   **New Default:** `{app.namespace}-{queueId}` (requires `app.namespace` to be configured for backward compatibility)
*   **Environment Variable:** `GOSO_CLOUD_AWS_SQS_CLIENTS_DEFAULT_NAMING_QUEUE_PATTERN`

*   **Delimiter Config Key:** `cloud.aws.sqs.clients.default.naming.queue_delimiter`
*   **Delimiter Default:** `-`
*   **Delimiter Environment Variable:** `GOSO_CLOUD_AWS_SQS_CLIENTS_DEFAULT_NAMING_QUEUE_DELIMITER`

**Note:** The new default uses `{app.namespace}`. To maintain backward compatibility, configure:
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

### SNS Topic Naming

*   **Config Key:** `cloud.aws.sns.clients.default.naming.topic_pattern`
*   **Old Default:** `{project}-{env}-{family}-{group}-{topicId}`
*   **New Default:** `{app.namespace}-{topicId}` (requires `app.namespace` to be configured for backward compatibility)
*   **Environment Variable:** `GOSO_CLOUD_AWS_SNS_CLIENTS_DEFAULT_NAMING_TOPIC_PATTERN`

*   **Delimiter Config Key:** `cloud.aws.sns.clients.default.naming.topic_delimiter`
*   **Delimiter Default:** `-`
*   **Delimiter Environment Variable:** `GOSO_CLOUD_AWS_SNS_CLIENTS_DEFAULT_NAMING_TOPIC_DELIMITER`

**Note:** The new default uses `{app.namespace}`. To maintain backward compatibility, configure:
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

### Kinesis Naming

#### Stream Pattern

*   **Config Key:** `cloud.aws.kinesis.clients.default.naming.stream_pattern` (Renamed from `pattern`)
*   **Old Default:** `{project}-{env}-{family}-{group}-{streamName}`
*   **New Default:** `{app.namespace}-{streamName}` (requires `app.namespace` to be configured for backward compatibility)
*   **Environment Variable:** `GOSO_CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_STREAM_PATTERN`

*   **Delimiter Config Key:** `cloud.aws.kinesis.clients.default.naming.stream_delimiter`
*   **Delimiter Default:** `-`
*   **Delimiter Environment Variable:** `GOSO_CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_STREAM_DELIMITER`

**Note:** The new default uses `{app.namespace}`. To maintain backward compatibility, configure:
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

#### Metadata Table Pattern

*   **Config Key:** `cloud.aws.kinesis.clients.default.naming.metadata_table_pattern`
*   **Old Default:** `{app.env}-kinsumer-metadata`
*   **New Default:** `{app.namespace}-kinsumer-metadata` (requires `app.namespace` to be configured for backward compatibility)
*   **Environment Variable:** `GOSO_CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_METADATA_TABLE_PATTERN`

*   **Delimiter Config Key:** `cloud.aws.kinesis.clients.default.naming.metadata_table_delimiter`
*   **Delimiter Default:** `-`
*   **Delimiter Environment Variable:** `GOSO_CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_METADATA_TABLE_DELIMITER`

**Important Change:** The metadata table pattern now uses `{app.namespace}` instead of just `{app.env}`. For backward compatibility, configure:
```yaml
app:
  namespace: "{app.env}"  # To match old default of {app.env}-kinsumer-metadata
```

Or use the full namespace pattern if you want the extended hierarchy:
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

#### Metadata Namespace Pattern (New Feature)

*   **Config Key:** `cloud.aws.kinesis.clients.default.naming.metadata_namespace_pattern`
*   **Old Default:** Not configurable (hardcoded based on app identity)
*   **New Default:** `{app.namespace}-{app.name}`
*   **Environment Variable:** `GOSO_CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_METADATA_NAMESPACE_PATTERN`

*   **Delimiter Config Key:** `cloud.aws.kinesis.clients.default.naming.metadata_namespace_delimiter`
*   **Delimiter Default:** `-`
*   **Delimiter Environment Variable:** `GOSO_CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_METADATA_NAMESPACE_DELIMITER`

**Note:** This new pattern controls the namespace prefix used within the metadata table for organizing client and checkpoint records. It allows multiple applications to share the same metadata table. For backward compatibility with previous internal naming:
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

### Kafka Naming

**Topic Pattern:**
*   **Config Key:** `kafka.naming.topic_pattern`
*   **Old Default:** `{project}-{env}-{family}-{group}-{topicId}`
*   **New Default:** `{app.namespace}-{topicId}` (requires `app.namespace` to be configured for backward compatibility)
*   **Environment Variable:** `GOSO_KAFKA_NAMING_TOPIC_PATTERN`

*   **Delimiter Config Key:** `kafka.naming.topic_delimiter`
*   **Delimiter Default:** `-`
*   **Delimiter Environment Variable:** `GOSO_KAFKA_NAMING_TOPIC_DELIMITER`

**Note:** The new default uses `{app.namespace}`. To maintain backward compatibility, configure:
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

**Consumer Group Pattern:**
*   **Config Key:** `kafka.naming.group_pattern`
*   **Old Default:** `{project}-{env}-{family}-{group}-{app}-{groupId}`
*   **New Default:** `{app.namespace}-{app.name}-{groupId}` (requires `app.namespace` to be configured for backward compatibility)
*   **Environment Variable:** `GOSO_KAFKA_NAMING_GROUP_PATTERN`

*   **Delimiter Config Key:** `kafka.naming.group_delimiter`
*   **Delimiter Default:** `-`
*   **Delimiter Environment Variable:** `GOSO_KAFKA_NAMING_GROUP_DELIMITER`

**Note:** The new default uses `{app.namespace}`. To maintain backward compatibility, configure:
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

### DynamoDB Table Naming

*   **Config Key:** `cloud.aws.dynamodb.clients.default.naming.table_pattern`
*   **Old Default:** `{project}-{env}-{family}-{group}-{modelId}`
*   **New Default:** `{app.namespace}-{name}` (requires `app.namespace` to be configured for backward compatibility)
*   **Environment Variable:** `GOSO_CLOUD_AWS_DYNAMODB_CLIENTS_DEFAULT_NAMING_TABLE_PATTERN`

*   **Delimiter Config Key:** `cloud.aws.dynamodb.clients.default.naming.delimiter`
*   **Delimiter Default:** `-`
*   **Delimiter Environment Variable:** `GOSO_CLOUD_AWS_DYNAMODB_CLIENTS_DEFAULT_NAMING_DELIMITER`

**Note:** The new default uses `{app.namespace}`. To maintain backward compatibility, configure:
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

### Metrics Naming

#### CloudWatch Namespace

*   **Config Key:** `metric.writer_settings.cloudwatch.naming.namespace_pattern`
*   **Old Default:** `{project}/{env}/{family}/{group}-{app}`
*   **New Default:** `{app.namespace}-{app.name}` (requires `app.namespace` to be configured for backward compatibility)
*   **Environment Variable:** `GOSO_METRIC_WRITER_SETTINGS_CLOUDWATCH_NAMING_NAMESPACE_PATTERN`

*   **Delimiter Config Key:** `metric.writer_settings.cloudwatch.naming.namespace_delimiter`
*   **Delimiter Default:** `/`
*   **Delimiter Environment Variable:** `GOSO_METRIC_WRITER_SETTINGS_CLOUDWATCH_NAMING_NAMESPACE_DELIMITER`

**Important:** CloudWatch uses `/` as the delimiter (not `-`), creating hierarchical namespaces in AWS Console.

**Note:** To maintain backward compatibility with the previous CloudWatch namespace format (`project/env/family/group-app`), configure:
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

This will result in namespaces like: `my-project/production/platform/core-my-app`

#### Prometheus Namespace

*   **Config Key:** `metric.writer_settings.prometheus.naming.namespace_pattern`
*   **Old Default:** Not explicitly documented (likely followed similar pattern)
*   **New Default:** `{app.namespace}-{app.name}` (requires `app.namespace` to be configured for backward compatibility)
*   **Environment Variable:** `GOSO_METRIC_WRITER_SETTINGS_PROMETHEUS_NAMING_NAMESPACE_PATTERN`

*   **Delimiter Config Key:** `metric.writer_settings.prometheus.naming.namespace_delimiter`
*   **Delimiter Default:** `_` (underscores, following Prometheus naming conventions)
*   **Delimiter Environment Variable:** `GOSO_METRIC_WRITER_SETTINGS_PROMETHEUS_NAMING_NAMESPACE_DELIMITER`

**Important:** Prometheus uses `_` (underscores) as the delimiter, which is standard for Prometheus metric naming.

**Note:** To maintain backward compatibility, configure:
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

This will result in metric prefixes like: `my_project_production_platform_core_my_app`

### Redis Key Prefix
*   **Config Key:** `redis.clients.default.naming.key_pattern`
*   **Old Default:** N/A (Feature did not exist)
*   **New Default:** `{key}` (No prefix)
*   **Environment Variable:** `GOSO_REDIS_CLIENTS_DEFAULT_NAMING_KEY_PATTERN`

*   **Delimiter Config Key:** `redis.clients.default.naming.key_delimiter`
*   **Delimiter Default:** `-`
*   **Delimiter Environment Variable:** `GOSO_REDIS_CLIENTS_DEFAULT_NAMING_KEY_DELIMITER`

**Note:** Redis also has an address pattern with delimiter:
*   **Address Delimiter Config Key:** `redis.clients.default.naming.address_delimiter`
*   **Address Delimiter Default:** `.` (dots)
*   **Address Delimiter Environment Variable:** `GOSO_REDIS_CLIENTS_DEFAULT_NAMING_ADDRESS_DELIMITER`

### KvStore Redis Key Pattern

The `kvstore` package configures Redis key patterns for each key-value store. The default pattern has changed to use `{app.namespace}`.

*   **Config Key:** `kvstore.<name>.redis.key_pattern` (per-store) or `kvstore.default.redis.key_pattern` (global default)
*   **Old Default:** `{app.tags.project}-{app.tags.family}-{app.tags.group}-kvstore-{store}-{key}`
*   **New Default:** `{app.namespace}-kvstore-{store}-{key}` (requires `app.namespace` to be configured for backward compatibility)

| Macro | Description |
|-------|-------------|
| `{store}` | The kvstore name (expanded before passing to Redis) |
| `{key}` | The specific key being accessed |

**Note:** The `{store}` placeholder is resolved by the kvstore config postprocessor before the pattern is passed to the underlying Redis client. The remaining placeholders are resolved by the Redis client's naming system.

**Migration:** To maintain backward compatibility with the previous key format, configure:
```yaml
app:
  namespace: "{app.tags.project}.{app.tags.family}.{app.tags.group}"
```

This will produce keys like: `my-project-my-family-my-group-kvstore-mystore-somekey` (same as the old default).

### Tracing Service Name

*   **Config Key:** `tracing.naming.pattern`
*   **Old Default:** `{project}-{env}-{family}-{group}-{app}`
*   **New Default:** `{app.namespace}-{app.name}` (requires `app.namespace` to be configured for backward compatibility)
*   **Environment Variable:** `GOSO_TRACING_NAMING_PATTERN`

*   **Delimiter Config Key:** `tracing.naming.delimiter`
*   **Delimiter Default:** `-`
*   **Delimiter Environment Variable:** `GOSO_TRACING_NAMING_DELIMITER`

**Note:** The new default uses `{app.namespace}`. To maintain backward compatibility, configure:
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}"
```

#### AWS X-Ray Daemon SRV Lookup (Optional)

If you use DNS SRV-based service discovery for the X-Ray daemon (`tracing.xray.addr_type: srv`), there is an additional naming pattern:

*   **Config Key:** `tracing.xray.srv_naming.pattern`
*   **Old Default:** `xray.{app.env}.{app.tags.family}` (implied from old SRV pattern logic)
*   **New Default:** `xray.{app.namespace}`
*   **Environment Variable:** `GOSO_TRACING_XRAY_SRV_NAMING_PATTERN`

*   **Delimiter Config Key:** `tracing.xray.srv_naming.delimiter`
*   **Delimiter Default:** `.` (dots, appropriate for DNS names)
*   **Delimiter Environment Variable:** `GOSO_TRACING_XRAY_SRV_NAMING_DELIMITER`

**Note:** This only applies if you're using `tracing.xray.addr_type: srv` for DNS-based X-Ray daemon discovery. For backward compatibility with SRV lookups:
```yaml
app:
  namespace: "{app.env}.{app.tags.family}"  # Or your previous SRV pattern structure
```

### Blob / S3 Bucket Patterns
The `blob` package now delegates bucket naming to the `cloud.aws.s3` package. The configuration keys `blob.<store_name>.bucket_pattern` and `blob.default.bucket_pattern` have been **removed**.

*   **Config Key:** `cloud.aws.s3.clients.<client_name>.naming.bucket_pattern`
*   **Old Default:** `{app.tags.project}-{app.env}-{app.tags.family}`
*   **New Default:** `{app.namespace}` (uses the configured namespace, see below)
*   **New Placeholder:** `{bucketId}` is now available (resolves to the blob store name)

*   **Delimiter Config Key:** `cloud.aws.s3.clients.<client_name>.naming.delimiter`
*   **Delimiter Default:** `-`
*   **Delimiter Environment Variable:** `GOSO_CLOUD_AWS_S3_CLIENTS_<CLIENT_NAME>_NAMING_DELIMITER`

**Migration:**
1. Move any custom bucket patterns to the new S3 naming config key. If you used `blob.default.bucket_pattern`, move it to `cloud.aws.s3.clients.default.naming.bucket_pattern`.
2. **Recommended:** Configure `app.namespace` to match your previous naming structure:
   ```yaml
   app:
     namespace: "{app.tags.project}.{app.env}.{app.tags.family}"
   ```
   This will ensure bucket names remain consistent with the old default pattern.

**Example:**
```yaml
# Old configuration (no longer supported)
blob:
  default:
    bucket_pattern: "{project}-{env}-data"

# New configuration
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}"
  
cloud:
  aws:
    s3:
      clients:
        default:
          naming:
            # Option 1: Use namespace (results in same pattern as before)
            bucket_pattern: "{app.namespace}-data"
            
            # Option 2: Define pattern explicitly
            bucket_pattern: "{app.tags.project}-{app.env}-data"
```

**Example Env Var Usage:**

```bash
# Override SQS naming to just use env and queueId
export GOSO_CLOUD_AWS_SQS_CLIENTS_DEFAULT_NAMING_QUEUE_PATTERN="{app.env}-{queueId}"

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

### mdlsub Publisher Configuration

The `PublisherSettings` struct has changed: `mdl.ModelId` is no longer an **embedded** (anonymous) field — it is now a **named field** with the config tag `model_id`. This means all ModelId-related configuration keys must be nested under a `model_id` key.

**Old:**
```yaml
mdlsub:
  publishers:
    my-model:
      name: my-model
      project: my-project
      family: my-family
      group: my-group
      output_type: sns
```

**New:**
```yaml
mdlsub:
  publishers:
    my-model:
      model_id:
        name: my-model          # optional: defaults to the publisher map key
        tags:
          project: my-project
          family: my-family
          group: my-group
      output_type: sns
```

**Key differences:**
1. Fields like `project`, `family`, `group` that were previously top-level under the publisher are now nested under `model_id.tags`.
2. The `name` field moves under `model_id.name` (still defaults to the publisher key name if omitted).
3. If you construct `PublisherSettings` in Go code, update from the embedded style to the named field:

```go
// OLD
settings := &mdlsub.PublisherSettings{
    ModelId: mdl.ModelId{
        Project: "my-project",
        Name:    "my-model",
    },
    OutputType: "sns",
}

// NEW
settings := &mdlsub.PublisherSettings{
    ModelId: mdl.ModelId{
        Name: "my-model",
        Tags: map[string]string{
            "project": "my-project",
        },
    },
    OutputType: "sns",
}
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
