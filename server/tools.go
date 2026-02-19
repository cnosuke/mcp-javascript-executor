package server

import (
	"context"
	"encoding/json"

	"github.com/cnosuke/mcp-javascript-executor/executor"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// RegisterAllTools - Register all tools with the MCP server
func RegisterAllTools(s *mcpserver.MCPServer, exec *executor.Executor) error {
	if err := registerExecuteJavaScriptTool(s, exec); err != nil {
		return err
	}
	return nil
}

func registerExecuteJavaScriptTool(s *mcpserver.MCPServer, exec *executor.Executor) error {
	zap.S().Debugw("registering execute_javascript tool")

	tool := mcp.NewTool("execute_javascript",
		mcp.WithDescription(`Execute JavaScript code safely in a sandboxed environment (ECMAScript 5.1, goja).

Supported: var declarations, regular functions, Math methods, JSON.stringify/parse, Array methods.
Constraints: 200ms timeout, 1024 max call stack, 10KB max code size, no I/O or network.`),
		mcp.WithString("code",
			mcp.Required(),
			mcp.Description(`JavaScript code to execute (ECMAScript 5.1 compatible, max 10KB).
Use JSON.stringify() to return complex objects. The last expression value is returned.
Examples: "100 + 200", "Math.sqrt(144)", "JSON.stringify({a:1+2})"`),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		code, err := request.RequireString("code")
		if err != nil {
			zap.S().Debugw("missing required parameter 'code'", "error", err)
			errResp := map[string]interface{}{
				"success": false,
				"error": map[string]interface{}{
					"type":    "ValidationError",
					"message": "parameter 'code' is required",
				},
			}
			jsonBytes, _ := json.Marshal(errResp)
			return mcp.NewToolResultText(string(jsonBytes)), nil
		}

		zap.S().Debugw("executing execute_javascript", "code_size", len(code))

		resultJSON, err := exec.Execute(ctx, code)
		if err != nil {
			zap.S().Errorw("failed to execute javascript", "error", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(resultJSON), nil
	})

	return nil
}
