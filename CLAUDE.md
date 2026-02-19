# mcp-javascript-executor

A sandboxed JavaScript executor MCP server written in Go.

## Architecture

```
main.go                  CLI entry (stdioserver / httpserver commands)
config/config.go         Configuration via koanf (YAML + env vars)
executor/executor.go     Core JS execution using goja
server/server.go         MCP server initialization
server/tools.go          execute_javascript tool registration
server/http.go           Streamable HTTP transport + graceful shutdown
server/middleware.go     Auth token (SHA-256 + ConstantTimeCompare) and CORS validation
logger/logger.go         zap logger initialization
internal/errors/wrap.go  Error wrapping utility
```

## Key Design Decisions

- **goja** (pure-Go ECMAScript engine): No CGO, no Node.js dependency. Each request gets a fresh VM (`goja.New()`) for isolation.
- **Timeout via goroutine + channel**: `vm.RunString()` runs in a goroutine. On timeout, `vm.Interrupt()` is called and `<-resultChan` waits for the goroutine to finish (prevents goroutine leak).
- **Sentinel error for timeout detection**: `errors.Is(err, errExecutionTimeout)` — not string matching.
- **SHA-256 auth token hashing**: `subtle.ConstantTimeCompare` requires equal-length slices; hashing prevents token length leakage.
- **No memory limit in goja**: goja has no memory limit API. Container/OS-level memory limits are required for production use. The 200ms timeout is the primary defense.

## Development Commands

```bash
make bin/mcp-javascript-executor   # Build
make test                          # Run tests (go test -v ./...)
go test -race ./...                # Run tests with race detector
make inspect                       # Run golangci-lint
make docker-build                  # Build Docker image
```

## Configuration

See `config.yml` for defaults. All keys are overridable via environment variables (e.g. `LOG_PATH`, `DEBUG`, `HTTP_AUTH_TOKEN`).

## Testing

- `executor/executor_test.go`: Unit tests for JS execution, timeout, stack overflow, goroutine leak detection
- `server/tools_test.go`: Integration tests using `server.HandleMessage()` with JSON-RPC messages

## Constraints

| Limit | Value |
|-------|-------|
| Max execution time | 200ms |
| Max code size | 10KB |
| Max call stack | 1024 |
| Max output size | 1MB |

## MCP Tool

- `execute_javascript`: Executes JS code, returns `{"success":bool,"result":string,"resultType":string,"executionTime":float64}`
