# DB-Repo Package Agent Guide

## Scope
- Repository abstraction on top of `pkg/db` using gosoline models.
- Handles CRUD, validation, change history, and notification hooks.
- Powers many `examples/` and integration tests that rely on SQL persistence.

## Key files
- `model.go`, `metadata.go` - describe model shape and table metadata.
- `repository.go`, `operation_repository_db.go` - core repository implementation.
- `change_history_*` - audit log support and middleware wiring.
- `notification_*` - publish DB changes to stream outputs.

## Common tasks
- Add repository features: extend `Repository` interface + implementation, then update mocks under `mocks/`.
- Modify change history: adjust manager/middleware and ensure tests under `change_history*_test.go` cover regression.
- Integrate new notification targets: modify `notification_publisher.go` and document required config keys (`db_repo.notification`).

## Testing
- `go test ./pkg/db-repo` is required; many tests rely on testify suites.
- When touching notifications, also run `go test ./pkg/mdlsub` to ensure downstream compatibility.

## Tips
- Use `mdl.ModelId` for naming to stay aligned with DynamoDB counterparts.
- Keep repository interfaces slim; cross-package consumers should rely on `pkg/db-repo/mocks` for isolation.
- Whenever you change default indexes or metadata, update fixture writers and docs in `examples/`.
