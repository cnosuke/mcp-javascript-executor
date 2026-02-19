# mcp-javascript-executor

A sandboxed JavaScript executor MCP server. Executes JavaScript code safely using [goja](https://github.com/dop251/goja) (ECMAScript 5.1).

## Features

- **Sandboxed execution**: No file I/O, network access, or async operations
- **Timeout protection**: 200ms execution limit
- **Stack overflow protection**: 1024 max call stack depth
- **Code size limit**: 10KB maximum
- **Transports**: STDIO and Streamable HTTP

## Installation

```bash
go install github.com/cnosuke/mcp-javascript-executor@latest
```

Or build from source:

```bash
make bin/mcp-javascript-executor
```

## Usage

### STDIO mode (for Claude Desktop)

```bash
./bin/mcp-javascript-executor stdioserver --config config.yml
```

### HTTP mode

```bash
./bin/mcp-javascript-executor httpserver --config config.yml
```

## Configuration

| Key | Environment Variable | Default | Description |
|-----|---------------------|---------|-------------|
| `log` | `LOG_PATH` | `""` | Log file path (empty = no logs) |
| `debug` | `DEBUG` | `false` | Debug mode |
| `http.binding` | `HTTP_BINDING` | `localhost:8080` | HTTP bind address |
| `http.endpoint_path` | `HTTP_ENDPOINT_PATH` | `/mcp` | MCP endpoint path |
| `http.heartbeat_seconds` | `HTTP_HEARTBEAT_SECONDS` | `30` | Heartbeat interval (seconds) |
| `http.auth_token` | `HTTP_AUTH_TOKEN` | `""` | Bearer auth token (empty = no auth) |
| `http.allowed_origins` | `HTTP_ALLOWED_ORIGINS` | `[]` | CORS allowed origins (comma-separated) |

## Claude Desktop Configuration

```json
{
  "mcpServers": {
    "javascript-executor": {
      "command": "/path/to/mcp-javascript-executor",
      "args": ["stdioserver", "--config", "/path/to/config.yml"]
    }
  }
}
```

## Tool: `execute_javascript`

Executes JavaScript code in a sandboxed ECMAScript 5.1 environment.

**Parameters:**
- `code` (string, required): JavaScript code to execute (max 10KB)

**Success response:**
```json
{
  "success": true,
  "result": "300",
  "resultType": "number",
  "executionTime": 0.5
}
```

**Error response:**
```json
{
  "success": false,
  "error": {
    "type": "TimeoutError",
    "message": "Script execution timed out after 200ms"
  },
  "suggestion": "Execution exceeded 200ms. Check for infinite loops or overly complex calculations."
}
```

**Result types:** `number`, `string`, `boolean`, `array`, `object`, `null`, `unknown`

**Error types:** `TimeoutError`, `StackOverflowError`, `SyntaxError`, `RuntimeError`, `ValidationError`, `UnknownError`

## Examples

```javascript
// Basic calculation
100 + 200
// => {"success":true,"result":"300","resultType":"number","executionTime":0.1}

// Array reduction
[1,2,3].reduce(function(a,b){return a+b},0)
// => {"success":true,"result":"6","resultType":"number","executionTime":0.2}

// Complex calculation with JSON output
var principal = 1000000;
var rate = 0.05;
var years = 10;
JSON.stringify({
  finalAmount: Math.floor(principal * Math.pow(1 + rate, years)),
  profit: Math.floor(principal * Math.pow(1 + rate, years) - principal)
});
// => {"success":true,"result":"{\"finalAmount\":1628894,\"profit\":628894}","resultType":"string","executionTime":0.3}
```
