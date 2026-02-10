# Naming Patterns in Gosoline

Gosoline uses a unified, configuration-driven approach for naming resources (SQS queues, SNS topics, DynamoDB tables, etc.) and other identifiers. This system relies on `AppIdentity` configuration and a set of macro substitutions.

## Core Concepts

### AppIdentity
Every Gosoline application is identified by its **name**, **environment**, and a set of **tags**. These are defined in your `config.dist.yml` or `config.yml`:

```yaml
app:
  name: my-app
  env: dev
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}"
  tags:
    project: my-project
    family: my-family
    group: my-group
    team: my-team
```

### Global Macros
The following macros are available in almost all naming patterns. They resolve directly from the `AppIdentity`:

| Macro | Source Config | Description |
|-------|---------------|-------------|
| `{app.name}` | `app.name` | The application name |
| `{app.env}` | `app.env` | The environment (e.g., dev, prod) |
| `{app.namespace}` | `app.namespace` | A reusable namespace pattern (see below) |
| `{app.tags.<tag>}` | `app.tags.<tag>` | Any tag defined in `app.tags` |

*Examples:*
- `{app.tags.project}` resolves to `my-project`
- `{app.tags.cost_center}` resolves to the value of `app.tags.cost_center`
- With namespace `{app.tags.project}.{app.env}.{app.tags.family}` and delimiter `-`, `{app.namespace}` resolves to `my-project-dev-my-family`

### Namespace Pattern
The `app.namespace` configuration allows you to define a reusable namespace pattern that can be referenced in all resource naming patterns. This is useful for establishing a consistent naming hierarchy across your infrastructure.

**Config Key:** `app.namespace`
**Default:** Empty (no namespace defined)

**Format:**
- The namespace is defined as a pattern using dots (`.`) as delimiters between placeholders
- Any standard `{app.*}` and `{app.tags.*}` placeholders can be used
- When expanded, the dots are replaced with the service-specific delimiter (usually `-`)

**Example:**
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}"
  tags:
    project: logistics
    family: platform
  env: production

# When used in patterns with delimiter "-", {app.namespace} becomes: logistics-production-platform
# When used in patterns with delimiter "/", {app.namespace} becomes: logistics/production/platform
```

**Benefits:**
- Define your naming hierarchy once and reuse it everywhere
- Simplify resource patterns by using `{app.namespace}` instead of repeating the same placeholders
- Easy to change your naming convention across all resources in one place

### Pattern Resolution and Delimiters

Every resource naming pattern is paired with a **delimiter** configuration that controls how `{app.namespace}` is expanded. There are two resolution mechanisms:

#### Identity.Format (with Delimiter)
Most services use `identity.Format(pattern, delimiter)`:
- The delimiter replaces dots in the `{app.namespace}` pattern
- Examples: SQS, SNS, Kinesis, DynamoDB, S3, Kafka, Redis, CloudWatch
- Each service has a configurable delimiter (default is usually `-`)

**Example:**
```yaml
app:
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}"
  tags:
    project: myproject
    family: myfamily
  env: prod

cloud.aws.sqs.clients.default.naming:
  queue_pattern: "{app.namespace}-{queueId}"
  queue_delimiter: "-"  # Dots in namespace become dashes
  
# Result for queue "orders": myproject-prod-myfamily-orders
```

#### Config.FormatString (no Delimiter)
This resolution method is no longer used by any services in gosoline. All services now use `identity.Format()` with configurable delimiters.

## Service-Specific Patterns

Different services support additional macros specific to their context. You can customize the naming pattern for each service in your configuration.

### AWS SQS (Queues)
**Config Key:** `cloud.aws.sqs.clients.<client_name>.naming.queue_pattern`
**Default:** `{app.namespace}-{queueId}`

**Delimiter Config Key:** `cloud.aws.sqs.clients.<client_name>.naming.queue_delimiter`
**Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{queueId}` | The logical name of the queue provided in code |

### AWS SNS (Topics)
**Config Key:** `cloud.aws.sns.clients.<client_name>.naming.topic_pattern`
**Default:** `{app.namespace}-{topicId}`

**Delimiter Config Key:** `cloud.aws.sns.clients.<client_name>.naming.topic_delimiter`
**Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{topicId}` | The logical name of the topic provided in code |

### AWS Kinesis
Kinesis configuration supports naming for Streams, the DynamoDB Metadata table, and metadata namespace used by the Kinsumer.

#### Stream Naming

**Stream Config Key:** `cloud.aws.kinesis.clients.<client_name>.naming.stream_pattern`
**Stream Default:** `{app.namespace}-{streamName}`

**Stream Delimiter Config Key:** `cloud.aws.kinesis.clients.<client_name>.naming.stream_delimiter`
**Stream Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{streamName}` | The logical name of the stream provided in code |

#### Metadata Table Naming

**Metadata Table Config Key:** `cloud.aws.kinesis.clients.<client_name>.naming.metadata_table_pattern`
**Metadata Table Default:** `{app.namespace}-kinsumer-metadata`

**Metadata Table Delimiter Config Key:** `cloud.aws.kinesis.clients.<client_name>.naming.metadata_table_delimiter`
**Metadata Table Default Delimiter:** `-` (dashes)

**Note:** This pattern determines the DynamoDB table name used to store Kinsumer checkpoint and client registration data.

#### Metadata Namespace Naming

**Metadata Namespace Config Key:** `cloud.aws.kinesis.clients.<client_name>.naming.metadata_namespace_pattern`
**Metadata Namespace Default:** `{app.namespace}-{app.name}`

**Metadata Namespace Delimiter Config Key:** `cloud.aws.kinesis.clients.<client_name>.naming.metadata_namespace_delimiter`
**Metadata Namespace Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.namespace}` | Your configured namespace pattern |
| `{app.tags.<tag>}` | Any tag from the identity |

**Note:** This pattern is used as a namespace prefix within the metadata table for organizing client and checkpoint records. It allows multiple applications to share the same metadata table while maintaining isolation.

### Metrics

#### AWS CloudWatch (Metrics Namespace)
CloudWatch naming configures the **Namespace** under which metrics are published.

**Config Key:** `metric.writer_settings.cloudwatch.naming.namespace_pattern`
**Default:** `{app.namespace}-{app.name}`

**Delimiter Config Key:** `metric.writer_settings.cloudwatch.naming.namespace_delimiter`
**Default Delimiter:** `/` (forward slashes)

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.namespace}` | Your configured namespace pattern |
| `{app.tags.<tag>}` | Any tag from the identity |

**Note:** CloudWatch uses `/` as the default delimiter, which creates hierarchical namespaces in the AWS Console (e.g., `my-project/production/platform/my-app`).

#### Prometheus (Metrics Namespace)
Prometheus naming configures the **namespace prefix** for all metrics exposed via the Prometheus metrics endpoint.

**Config Key:** `metric.writer_settings.prometheus.naming.namespace_pattern`
**Default:** `{app.namespace}-{app.name}`

**Delimiter Config Key:** `metric.writer_settings.prometheus.naming.namespace_delimiter`
**Default Delimiter:** `_` (underscores)

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.namespace}` | Your configured namespace pattern |
| `{app.tags.<tag>}` | Any tag from the identity |

**Note:** Prometheus uses `_` (underscores) as the default delimiter, which is standard for Prometheus metric naming conventions (e.g., `my_project_prod_platform_my_app`).

### AWS DynamoDB (Tables)
DynamoDB table naming uses the standard `AppIdentity` macros plus `{name}` for the model name.

**Config Key:** `cloud.aws.dynamodb.clients.<client_name>.naming.table_pattern`
**Default:** `{app.namespace}-{name}`

**Delimiter Config Key:** `cloud.aws.dynamodb.clients.<client_name>.naming.table_delimiter`
**Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.tags.<tag>}` | Any tag from the model's identity |
| `{name}` | The model name |

**Note:** Unlike the Canonical Model ID, the table name pattern **does not** automatically append the name. You must include `{name}` in the pattern.

### DynamoDB Leader Election Table
The `conc/ddb` package uses a DynamoDB table for distributed leader election. It has its own naming pattern that is independent of the main DynamoDB table naming.

**Config Key:** `conc.ddb.leader_election.<name>.naming.table_pattern`
**Default:** `{app.tags.project}-{app.env}-{app.tags.family}-leader-elections`

**Delimiter Config Key:** `conc.ddb.leader_election.<name>.naming.table_delimiter`
**Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.namespace}` | Your configured namespace pattern |
| `{app.tags.<tag>}` | Any tag from the identity |

**Note:** This table stores leader election state (group ID, member ID, and lease expiration). The table is shared across modules that use DDB-based leader election (e.g., metric calculator, kinsumer autoscale).

### Metric Calculator DynamoDB Table
The metric calculator module uses a DynamoDB table for leader election among calculator instances. It has its own naming pattern.

**Config Key:** `metric.calculator.dynamodb.naming.table_pattern`
**Default:** `{app.env}-metric-calculator-leaders`

**Delimiter Config Key:** `metric.calculator.dynamodb.naming.table_delimiter`
**Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.namespace}` | Your configured namespace pattern |
| `{app.tags.<tag>}` | Any tag from the identity |

**Note:** This pattern is passed to the DDB leader election module. The group ID for metric calculator leader election is constructed as `{namespace}-{app.name}`, where `{namespace}` is the expanded `app.namespace` with `-` as delimiter.

### Kinsumer Autoscale DynamoDB Table
The kinsumer autoscale module uses a DynamoDB table for leader election. It has its own naming pattern.

**Config Key:** `kinsumer.autoscale.dynamodb.naming.pattern`
**Default:** `{app.env}-kinsumer-autoscale-leaders`

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.namespace}` | Your configured namespace pattern |
| `{app.tags.<tag>}` | Any tag from the identity |

**Note:** This pattern is passed to the DDB leader election module. The group ID for kinsumer autoscale leader election is constructed as `{namespace}-{app.name}`, where `{namespace}` is the expanded `app.namespace` with `-` as delimiter.

### AWS S3 Buckets
S3 bucket naming is used by the `blob` package and other S3-dependent components.

**Config Key:** `cloud.aws.s3.clients.<client_name>.naming.bucket_pattern`
**Default:** `{app.namespace}` (uses your configured namespace with the bucket delimiter)

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.namespace}` | Your configured namespace pattern |
| `{app.tags.<tag>}` | Any tag from the identity |
| `{bucketId}` | The bucket ID (e.g. the blob store name) |

**Delimiter Config Key:** `cloud.aws.s3.clients.<client_name>.naming.delimiter`
**Default Delimiter:** `-` (dashes)

**Notes:** 
- For the `blob` package, you can still override the bucket name explicitly using `blob.<store_name>.bucket`
- If no namespace is configured, `{app.namespace}` expands to an empty string
- The delimiter determines how dots in the namespace pattern are replaced (e.g., `.` → `-`)

### ModelId Domain Pattern
Resources that use `ModelId` but are not tied to a specific service client (like canonical message routing keys) use the domain pattern.

**Config Key:** `app.model_id.domain_pattern`
**Recommended:** `{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}`

This pattern works similarly to DynamoDB naming:
1.  It resolves the pattern using the model's identity tags.
2.  It automatically appends the model name as the final segment (separated by a dot).

**Strict Delimiter Constraint:**
Unlike other naming patterns, the `ModelId` domain pattern **must** use dots (`.`) as delimiters between placeholders. Dashes, underscores, or static text are **not permitted**. This strict format allows the system to parse a string representation back into a `ModelId` object.

*   ✅ `{app.tags.project}.{app.env}`
*   ❌ `{app.tags.project}-{app.env}` (Invalid delimiter)

**Example:**
*   Pattern: `{app.tags.project}.{app.env}`
*   Model Identity: `env=prod`, `project=logistics`, `name=Shipment`
*   Result: `logistics.prod.Shipment`

### Kafka
Kafka supports naming for both Topics and Consumer Groups.

**Topic Config Key:** `kafka.naming.topic_pattern`
**Topic Default:** `{app.namespace}-{topicId}`

**Topic Delimiter Config Key:** `kafka.naming.topic_delimiter`
**Topic Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{topicId}` | The logical topic name |

**Consumer Group Config Key:** `kafka.naming.group_pattern`
**Group Default:** `{app.namespace}-{app.name}-{groupId}`

**Group Delimiter Config Key:** `kafka.naming.group_delimiter`
**Group Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{groupId}` | The logical consumer group ID |

### Redis
Redis has patterns for both the server address (for service discovery) and key namespacing.

**Address Config Key:** `redis.<client_name>.naming.address_pattern`
**Address Default:** `{name}.{app.tags.group}.redis.{app.env}.{app.tags.family}`

**Address Delimiter Config Key:** `redis.<client_name>.naming.address_delimiter`
**Address Default Delimiter:** `.` (dots)

| Macro | Description |
|-------|-------------|
| `{name}` | The client name (e.g., "default", "cache") |

**Key Config Key:** `redis.<client_name>.naming.key_pattern`
**Key Default:** `{key}` (often configured to include namespaces like `{app.namespace}-{app.name}-{key}`)

**Key Delimiter Config Key:** `redis.<client_name>.naming.key_delimiter`
**Key Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{key}` | The specific key being accessed |

### KvStore (Redis Key Pattern)
The `kvstore` package uses a key pattern to namespace Redis keys for each key-value store. The `{store}` placeholder is expanded by `config_postprocessor_redis.go` before the pattern is passed to the underlying Redis client configuration.

**Config Key:** `kvstore.<name>.redis.key_pattern` (per-store) or `kvstore.default.redis.key_pattern` (global default)
**Default:** `{app.namespace}-kvstore-{store}-{key}`

**Delimiter Config Key:** Inherits from the underlying Redis client's `naming.key_delimiter`
**Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{store}` | The name of the kvstore (expanded before passing to Redis) |
| `{key}` | The specific key being accessed |
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.namespace}` | Your configured namespace pattern |
| `{app.tags.<tag>}` | Any tag from the identity |

**Note:** The `{store}` placeholder is resolved by the kvstore config postprocessor, not by `AppIdentity.Format()`. The remaining placeholders (`{key}`, `{app.*}`) are resolved by the Redis client's naming system.

### Tracing
Tracing service naming configures the service name used in distributed tracing systems like AWS X-Ray and OpenTelemetry.

**Service Name Config Key:** `tracing.naming.pattern`
**Service Name Default:** `{app.namespace}-{app.name}`

**Service Name Delimiter Config Key:** `tracing.naming.delimiter`
**Service Name Default Delimiter:** `-` (dashes)

| Macro | Description |
|-------|-------------|
| `{app.env}` | Environment |
| `{app.name}` | Application name |
| `{app.namespace}` | Your configured namespace pattern |
| `{app.tags.<tag>}` | Any tag from the identity |

#### AWS X-Ray Daemon SRV Lookup

When using AWS X-Ray with DNS SRV-based service discovery (`tracing.xray.addr_type: srv`), you can configure the SRV record name pattern:

**SRV Pattern Config Key:** `tracing.xray.srv_naming.pattern`
**SRV Pattern Default:** `xray.{app.namespace}`

**SRV Delimiter Config Key:** `tracing.xray.srv_naming.delimiter`
**SRV Delimiter Default:** `.` (dots)

**Notes:**
- This pattern is only used when `tracing.xray.addr_type` is set to `srv` (DNS SRV lookup)
- The delimiter defaults to `.` (dots) which is appropriate for DNS names
- If `tracing.xray.add_value` is explicitly configured, this pattern is ignored

**Example:**
```yaml
app:
  namespace: "{app.tags.project}.{app.env}"
  tags:
    project: myproject
  env: prod

tracing:
  xray:
    addr_type: srv  # Enable DNS SRV lookup
    srv_naming:
      pattern: "xray.{app.namespace}"
      delimiter: "."
      
# SRV lookup will query: xray.myproject.prod
```

## Configuration Example

Here is how you might configure these patterns in your `config.yml` to enforce a company-wide naming convention:

```yaml
app:
  name: order-service
  env: production
  namespace: "{app.tags.project}.{app.env}.{app.tags.family}"
  tags:
    project: logistics
    family: platform
    region: eu-central-1
    team: backend

cloud:
  aws:
    sqs:
      clients:
        default:
          naming:
            # Using namespace: logistics-production-platform-myqueue
            queue_pattern: "{app.namespace}-{queueId}"
            
    s3:
      clients:
        default:
          naming:
            # Using namespace with bucketId: logistics-production-platform-documents
            bucket_pattern: "{app.namespace}-{bucketId}"
            
    sns:
      clients:
        default:
          naming:
            # Custom pattern: eu-central-1-backend-logistics-production-mytopic
            topic_pattern: "{app.tags.region}-{app.tags.team}-{app.namespace}-{topicId}"

metric:
  writer_settings:
    cloudwatch:
      naming:
        # Using namespace with slash delimiter: logistics/production/platform-order-service
        namespace_pattern: "{app.namespace}-{app.name}"
        namespace_delimiter: "/"
```

**Benefits of using `app.namespace`:**
- Define your naming hierarchy once (`{app.tags.project}.{app.env}.{app.tags.family}`)
- Reuse it across all resources with `{app.namespace}`
- Change the hierarchy in one place to update all resource names
- Keep resource-specific patterns simple and focused
