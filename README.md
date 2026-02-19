# mcp-javascript-executor

A Go-based MCP server that executes JavaScript code in a sandboxed environment, allowing MCP clients (e.g., Claude Desktop) to safely run JavaScript calculations and data transformations.

## Features

- **Sandboxed execution**: No file I/O, network access, or system calls — powered by [goja](https://github.com/dop251/goja)
- **Timeout protection**: 200ms execution limit
- **Stack overflow protection**: 1024 max call stack depth
- **Code size limit**: 10KB maximum input
- **Transport options**: Supports both STDIO and Streamable HTTP transports

## Requirements

- Docker (recommended)

For local development:

- Go 1.24 or later

## Using with Docker (Recommended)

```bash
docker pull cnosuke/mcp-javascript-executor:latest

docker run -i --rm cnosuke/mcp-javascript-executor:latest
```

### Using with Claude Desktop (Docker)

Add an entry to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "javascript-executor": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "cnosuke/mcp-javascript-executor:latest"]
    }
  }
}
```

## Building and Running (Go Binary)

```bash
# Build the server
make bin/mcp-javascript-executor
```

### STDIO Transport

Run the server with STDIO transport (used with Claude Desktop):

```bash
./bin/mcp-javascript-executor stdioserver --config=config.yml
```

### HTTP Transport (Streamable HTTP)

Run the server with Streamable HTTP transport:

```bash
./bin/mcp-javascript-executor httpserver --config=config.yml
```

### Using with Claude Desktop (Go Binary)

Add an entry to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "javascript-executor": {
      "command": "./bin/mcp-javascript-executor",
      "args": ["stdioserver"],
      "env": {
        "LOG_PATH": "mcp-javascript-executor.log",
        "DEBUG": "false"
      }
    }
  }
}
```

## Configuration

The server is configured via a YAML file (default: `config.yml`):

```yaml
log: 'path/to/mcp-javascript-executor.log' # Log file path, if empty no log will be produced
debug: false # Enable debug mode for verbose logging

http:
  binding: "localhost:8080"       # HTTP bind address
  endpoint_path: "/mcp"           # MCP endpoint path
  heartbeat_seconds: 30           # SSE heartbeat interval
  auth_token: ""                  # Bearer auth token (empty = no auth)
  allowed_origins: []             # Allowed CORS origins
```

You can override configurations using environment variables:

| Environment Variable     | Config Key                 | Default          | Description                             |
|--------------------------|----------------------------|------------------|-----------------------------------------|
| `LOG_PATH`               | `log`                      | `""`             | Log file path (empty = no logs)         |
| `DEBUG`                  | `debug`                    | `false`          | Enable debug mode                       |
| `HTTP_BINDING`           | `http.binding`             | `localhost:8080` | HTTP bind address                       |
| `HTTP_ENDPOINT_PATH`     | `http.endpoint_path`       | `/mcp`           | MCP endpoint path                       |
| `HTTP_HEARTBEAT_SECONDS` | `http.heartbeat_seconds`   | `30`             | Heartbeat interval (seconds)            |
| `HTTP_AUTH_TOKEN`        | `http.auth_token`          | `""`             | Bearer auth token (empty = no auth)     |
| `HTTP_ALLOWED_ORIGINS`   | `http.allowed_origins`     | `[]`             | Allowed CORS origins (comma-separated)  |

## Logging

Logging behavior is controlled through configuration:

- If `log` is set in the config file, logs will be written to the specified file
- If `log` is empty, no logs will be produced
- Set `debug: true` for more verbose logging

## MCP Server Usage

MCP clients interact with the server by sending JSON-RPC requests to execute tools. The following MCP tools are supported:

- `execute_javascript`: Executes JavaScript code in a sandboxed environment and returns the result

### Tool: `execute_javascript`

Executes JavaScript code safely with strict resource limits.

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

**Error types:** `ValidationError`, `TimeoutError`, `StackOverflowError`, `SyntaxError`, `RuntimeError`, `OutputTooLargeError`, `UnknownError`

### Examples

```javascript
// Basic calculation
100 + 200
// => {"success":true,"result":"300","resultType":"number","executionTime":0.1}

// Array operations
[1,2,3].reduce(function(a,b){return a+b},0)
// => {"success":true,"result":"6","resultType":"number","executionTime":0.2}

// Compound interest calculation
var principal = 1000000;
var rate = 0.05;
var years = 10;
JSON.stringify({
  finalAmount: Math.floor(principal * Math.pow(1 + rate, years)),
  profit: Math.floor(principal * Math.pow(1 + rate, years) - principal)
});
// => {"success":true,"result":"{\"finalAmount\":1628894,\"profit\":628894}","resultType":"string","executionTime":0.3}

// Pi approximation (BBP formula)
var pi = 0;
for (var k = 0; k < 15; k++) {
  var p8k = Math.pow(16, k);
  pi += (1/p8k)*(4/(8*k+1)-2/(8*k+4)-1/(8*k+5)-1/(8*k+6));
}
pi
// => {"success":true,"result":"3.141592653589793","resultType":"number","executionTime":0.2}
```

## Command-Line Reference

```
COMMANDS:
   stdioserver, stdio, s  Run MCP server with STDIO transport
   httpserver, http       Run MCP server with Streamable HTTP transport

OPTIONS (both commands):
   --config value, -c value  Path to the configuration file (default: "config.yml")
```

## Limitations

- **Memory**: goja has no built-in memory limit. A script can consume significant memory within the 200ms timeout window. For production deployments, set memory limits at the container or OS level (e.g., Docker `--memory` flag).
- **ECMAScript version**: Supports ES5.1 and many ES2015+ features via goja. However, async/await, Promises, and Node.js built-ins (fs, http, etc.) are not available.
- **No persistent state**: Each execution runs in a fresh JavaScript VM with no shared state between calls.

## Contributing

Contributions are welcome! Please fork the repository and submit pull requests for improvements or bug fixes. For major changes, open an issue first to discuss your ideas.

## License

This project is licensed under the MIT License.

Author: cnosuke ( x.com/cnosuke )
