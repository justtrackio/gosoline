# Blob Package Agent Guide

## Scope
- Houses the blob storage abstraction and S3 implementation.
- Provides `Store` interface for CRUD operations on objects.
- Handles batch operations, lifecycle management, and fixtures.

## Key files
- `store.go` - `Store` interface, S3 implementation, settings loading.
- `service.go` - `Service` struct for creating/checking buckets.
- `runner.go` - `BatchRunner` for asynchronous operations.
- `url_builder.go` - helper for generating absolute URLs to blobs.

## Common tasks
- Adding a new store: configure `blob.<name>` settings.
- Adjusting bucket naming: configure `blob.default.bucket_pattern` or `blob.<name>.bucket_pattern`.
- Implementing new storage backend: implement `Store` interface.

## Configuration

### Bucket Naming
Bucket naming can be configured per store or globally using patterns with `cfg.AppIdentity` placeholders.

**Priority:**
1. `blob.<name>.bucket` (explicit override)
2. `blob.<name>.bucket_pattern`
3. `blob.default.bucket_pattern`
4. Default pattern: `{app.tags.project}-{app.env}-{app.tags.family}`

**Supported placeholders:**
- `{app.env}` - Environment
- `{app.name}` - Application name
- `{app.tags.<key>}` - Any tag value (e.g., `{app.tags.project}`, `{app.tags.costCenter}`)

**Config examples:**
```yaml
# Global default pattern
blob.default.bucket_pattern: "{app.tags.project}-{app.env}-{app.name}-blobs"

# Store-specific pattern
blob.images.bucket_pattern: "{app.env}-images-{app.tags.region}"

# Explicit bucket name (ignores patterns)
blob.legacy.bucket: "my-legacy-bucket"
```

## Testing
- `go test ./pkg/blob` for unit tests.
- Integration tests require AWS credentials or LocalStack.

## Related packages
- `pkg/cloud/aws/s3` - low-level S3 client
- `pkg/cfg` - configuration and naming template support
