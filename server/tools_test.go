package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cnosuke/mcp-javascript-executor/executor"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func setupTestLogger(t *testing.T) {
	t.Helper()
	logger := zaptest.NewLogger(t)
	zap.ReplaceGlobals(logger)
}

func callTool(t *testing.T, s *mcpserver.MCPServer, code string) map[string]interface{} {
	t.Helper()
	params := map[string]interface{}{
		"name": "execute_javascript",
	}
	if code != "" {
		params["arguments"] = map[string]interface{}{"code": code}
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	msg := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":` + string(paramsJSON) + `}`)
	resp := s.HandleMessage(context.Background(), msg)

	jsonResp, ok := resp.(mcp.JSONRPCResponse)
	require.True(t, ok, "expected JSONRPCResponse, got %T", resp)

	resultJSON, err := json.Marshal(jsonResp.Result)
	require.NoError(t, err)

	var callResult map[string]interface{}
	require.NoError(t, json.Unmarshal(resultJSON, &callResult))

	// Extract text from content array
	content, ok := callResult["content"].([]interface{})
	require.True(t, ok, "expected content array")
	require.NotEmpty(t, content)

	firstContent, ok := content[0].(map[string]interface{})
	require.True(t, ok)

	text, ok := firstContent["text"].(string)
	require.True(t, ok, "expected text content")

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(text), &m))
	return m
}

func TestRegisterAllTools(t *testing.T) {
	setupTestLogger(t)

	s := mcpserver.NewMCPServer("test-server", "0.0.1")
	exec := executor.NewExecutor()

	err := RegisterAllTools(s, exec)
	assert.NoError(t, err)
}

func TestExecuteJavaScriptTool_Success(t *testing.T) {
	setupTestLogger(t)

	s := mcpserver.NewMCPServer("test-server", "0.0.1")
	exec := executor.NewExecutor()
	require.NoError(t, RegisterAllTools(s, exec))

	m := callTool(t, s, "100 + 200")
	assert.Equal(t, true, m["success"])
	assert.Equal(t, "300", m["result"])
	assert.Equal(t, "number", m["resultType"])
}

func TestExecuteJavaScriptTool_TimeoutError(t *testing.T) {
	setupTestLogger(t)

	s := mcpserver.NewMCPServer("test-server", "0.0.1")
	exec := executor.NewExecutor()
	require.NoError(t, RegisterAllTools(s, exec))

	m := callTool(t, s, "while(true){}")
	assert.Equal(t, false, m["success"])
	errObj := m["error"].(map[string]interface{})
	assert.Equal(t, "TimeoutError", errObj["type"])
}

func TestExecuteJavaScriptTool_SyntaxError(t *testing.T) {
	setupTestLogger(t)

	s := mcpserver.NewMCPServer("test-server", "0.0.1")
	exec := executor.NewExecutor()
	require.NoError(t, RegisterAllTools(s, exec))

	m := callTool(t, s, "if (")
	assert.Equal(t, false, m["success"])
	errObj := m["error"].(map[string]interface{})
	assert.Equal(t, "SyntaxError", errObj["type"])
}

func TestExecuteJavaScriptTool_MissingCode(t *testing.T) {
	setupTestLogger(t)

	s := mcpserver.NewMCPServer("test-server", "0.0.1")
	exec := executor.NewExecutor()
	require.NoError(t, RegisterAllTools(s, exec))

	// Send request without code parameter
	msg := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"execute_javascript","arguments":{}}}`)
	resp := s.HandleMessage(context.Background(), msg)

	jsonResp, ok := resp.(mcp.JSONRPCResponse)
	require.True(t, ok, "expected JSONRPCResponse, got %T", resp)

	resultJSON, err := json.Marshal(jsonResp.Result)
	require.NoError(t, err)

	var callResult map[string]interface{}
	require.NoError(t, json.Unmarshal(resultJSON, &callResult))

	content := callResult["content"].([]interface{})
	firstContent := content[0].(map[string]interface{})
	text := firstContent["text"].(string)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(text), &m))
	assert.Equal(t, false, m["success"])
	errObj := m["error"].(map[string]interface{})
	assert.Equal(t, "ValidationError", errObj["type"])
}
