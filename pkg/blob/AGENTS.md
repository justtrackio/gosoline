# Blob Package Agent Guide

## Scope
- Houses the blob storage abstraction and S3 implementation.
- Provides `Store` interface for CRUD operations on objects.
- Handles batch operations, lifecycle management, and fixtures.

## Key files
- `store.go` - `Store` interface and S3 implementation.
- `settings.go` - settings loading and bucket name resolution.
- `service.go` - `Service` struct for creating/checking buckets.
- `runner.go` - `BatchRunner` for asynchronous operations.
- `url_builder.go` - helper for generating absolute URLs to blobs.

## Common tasks
- Adding a new store: configure `blob.<name>` settings and register a `blob.ProvideBatchRunner("<name>")` kernel module. The store sends all operations (read, write, copy, delete) through shared channels that the `BatchRunner` drains — without it, operations deadlock.
- Adjusting bucket naming: configure `blob.default.bucket_pattern` or `blob.<name>.bucket_pattern`.
- Implementing new storage backend: implement `Store` interface.

## Configuration

### Bucket Naming
Bucket naming is handled by the `pkg/cloud/aws/s3` package naming settings. The `blob` package provides the store name as the `{bucketId}`.

**Priority:**
1. `blob.<name>.bucket` (explicit override)
2. `cloud.aws.s3.clients.<client>.naming.bucket_pattern`
3. `cloud.aws.s3.clients.default.naming.bucket_pattern`
4. Default pattern: `{app.namespace}`

**Supported placeholders:**
- `{app.env}` - Environment
- `{app.name}` - Application name
- `{app.tags.<key>}` - Any tag value
- `{bucketId}` - The name of the blob store

**Config examples:**
```yaml
# Global default pattern for all S3 clients (including blob stores)
cloud.aws.s3.clients.default.naming.bucket_pattern: "{app.tags.project}-{app.env}-{app.name}-blobs-{bucketId}"

# Store-specific S3 client pattern (if blob uses a specific client)
cloud.aws.s3.clients.my_store_client.naming.bucket_pattern: "{app.env}-images-{app.tags.region}"

# Explicit bucket name (ignores patterns)
blob.legacy.bucket: "my-legacy-bucket"
```

## Testing
- `go test ./pkg/blob` for unit tests.
- Integration tests require AWS credentials or LocalStack.

## Related packages
- `pkg/cloud/aws/s3` - low-level S3 client
- `pkg/cfg` - configuration and naming template support
