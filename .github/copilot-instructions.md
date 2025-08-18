# Gosoline Application Framework

**ALWAYS follow these instructions first and only fallback to additional search and context gathering if the information in these instructions is incomplete or found to be in error.**

Gosoline is a Golang-based application framework specialized for building microservices in the cloud. It provides tools for configuration, logging, structured code execution, HTTP requests, asynchronous message processing, integration testing, and AWS services integration.

## Working Effectively

### Bootstrap and Build Process
Execute these commands in order to set up and build the repository:

1. **Download dependencies** (5+ minutes initially):
   - `go get -v -t -d ./...`
   - NEVER CANCEL: Takes ~5 minutes on first run. Set timeout to 10+ minutes.

2. **Build the entire project** (2-3 minutes):
   - `go build -v ./...`
   - NEVER CANCEL: Takes 2m 22s to complete. Set timeout to 5+ minutes.

3. **Generate mocks** (1-2 minutes):
   - `go generate -run='mockery' ./...`
   - NEVER CANCEL: Takes 1m 4s to complete. Set timeout to 3+ minutes.

### Testing Process

1. **Unit tests** (2-3 minutes):
   - `go test ./...`
   - NEVER CANCEL: Takes 1m 54s to complete. Set timeout to 4+ minutes.

2. **Unit tests with race detector** (5-7 minutes):
   - `go test -race ./...`
   - NEVER CANCEL: Takes 4m 44s to complete. Set timeout to 8+ minutes.

3. **Integration tests** (40-60+ minutes):
   - Linux: `go test -tags integration,fixtures ./test/...`
   - macOS: First run `sudo ifconfig lo0 alias 172.17.0.1`, then `go test -tags integration,fixtures ./test/...`
   - NEVER CANCEL: Takes 40+ minutes to complete. Set timeout to 90+ minutes.
   - **Note**: Requires Docker for container-based testing. Some tests may fail in CI environments with Docker issues.

### Documentation Build Process

1. **Install dependencies** (30-60 seconds):
   - `cd docs && yarn install`
   - NEVER CANCEL: Takes ~40 seconds. Set timeout to 2+ minutes.

2. **Build documentation** (30 seconds):
   - `cd docs && yarn build`
   - NEVER CANCEL: Takes ~30 seconds. Set timeout to 2+ minutes.

3. **Serve documentation locally**:
   - `cd docs && yarn start` (development server)
   - `cd docs && yarn serve` (production build)

### Linting and Code Quality

1. **Format code**:
   - `gofumpt -w .` (if gofumpt is available)
   
2. **Generate mocks**:
   - `go generate -run='mockery' ./...`
   - **Always run this before committing if you modify interfaces**

3. **CI validation commands**:
   - The CI runs: mockery check, build, golangci-lint, unit tests, race tests, and integration tests
   - **Always ensure mockery is up to date** - CI will fail if mocks are outdated

## Validation Scenarios

**ALWAYS manually validate any changes with these scenarios:**

### Basic Application Testing
1. **Test simple application**:
   - `cd examples/application && go run main.go`
   - Should output "Hello World" and exit cleanly within 2 seconds
   - Verify logs show kernel startup and shutdown

### HTTP Server Testing
1. **Start HTTP server**:
   - `cd examples/httpserver/simple-handlers && go run .`
   - Should start on port 8088 and show Gin debug routes

2. **Test endpoints**:
   - Health check: `curl http://localhost:8088/health` → `{}`
   - JSON endpoint: `curl http://localhost:8088/json-from-struct` → `{"status":"success"}`
   - POST endpoint: `curl -H "Content-Type: application/json" -d '{"name":"test","message":"hello"}' http://localhost:8088/json-handler`
   - Should return: `{"message":"Thank you for submitting your message 'hello', we will handle it with care!"}`

3. **Stop server**: `Ctrl+C` or `pkill -f "simple-handlers"`

### Documentation Testing
1. **Build and verify docs**:
   - `cd docs && yarn build`
   - Should complete without errors and create `build/` directory
   - Optionally test with: `yarn serve`

## Repository Structure

### Key Directories
- `/pkg/` - Main Go packages (40+ packages including application, httpserver, cfg, log, etc.)
- `/test/` - Integration tests (blob, cloud, db, httpserver, etc.)
- `/examples/` - Working example applications for different use cases
- `/docs/` - Docusaurus documentation site
- `/.github/workflows/` - CI/CD configuration

### Important Files
- `go.mod` - Go module definition (uses Go 1.24)
- `.golangci.yml` - Linting configuration
- `.mockery.yml` - Mock generation configuration
- `.tool-versions` - asdf tool versions (Go 1.24.0, Node 20.15.1, etc.)
- `Makefile` - Contains embedmd command for documentation

### Package Organization
**Core packages** (frequently modified):
- `pkg/application/` - Application framework entry point
- `pkg/cfg/` - Configuration management
- `pkg/log/` - Logging infrastructure
- `pkg/httpserver/` - HTTP server and middleware
- `pkg/kernel/` - Application lifecycle management
- `pkg/stream/` - Message streaming and processing

**AWS integration packages**:
- `pkg/cloud/aws/` - AWS service integrations (S3, DynamoDB, SQS, SNS, etc.)

**Testing packages**:
- `pkg/test/` - Testing utilities and fixtures
- `test/` - Integration test suites

## Common Patterns and Conventions

### Configuration Files
- Applications require `config.dist.yml` with at least:
  ```yaml
  env: dev
  app_project: [project-name]
  app_family: [family-name]
  app_group: [group-name]
  app_name: [app-name]
  ```

### Module Creation
- Applications use `application.Run()` with module factories
- Modules implement `kernel.Module` interface with `Run(ctx context.Context) error`
- Use dependency injection via module factory functions

### Error Handling
- Check for build tag requirements (e.g., `integration,fixtures` for tests)
- Integration tests require Docker and may fail in restricted environments
- Some tests require specific network configuration on macOS

## Troubleshooting

### Build Issues
- **Mock generation fails**: Run `go generate -run='mockery' ./...` to regenerate
- **Integration tests fail**: Ensure Docker is running and accessible
- **macOS integration test issues**: Run `sudo ifconfig lo0 alias 172.17.0.1`

### Development Workflow
1. Make code changes
2. Run `go generate -run='mockery' ./...` if interfaces changed
3. Run `go build -v ./...` to ensure compilation
4. Run `go test ./...` for unit tests
5. Test with example applications for functional validation
6. Run integration tests if modifying core functionality

### Performance Expectations
- **Build times**: 2-3 minutes for full build
- **Test times**: 2 minutes unit tests, 5 minutes with race, 40+ minutes integration
- **Memory usage**: Integration tests may require significant Docker resources
- **Parallelism**: Integration tests use `-p 2` to limit Docker container conflicts

**Remember**: This is a mature, production-ready framework with extensive AWS integration. Always test both unit functionality and integration scenarios when making changes.