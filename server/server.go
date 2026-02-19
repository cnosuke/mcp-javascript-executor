package server

import (
	"context"

	"github.com/cnosuke/mcp-javascript-executor/config"
	"github.com/cnosuke/mcp-javascript-executor/executor"
	ierrors "github.com/cnosuke/mcp-javascript-executor/internal/errors"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// RunStdio - Execute the MCP server with STDIO transport
func RunStdio(cfg *config.Config, name string, version string, revision string) error {
	zap.S().Infow("starting MCP JavaScript Executor Server with STDIO transport")

	s, err := createMCPServer(cfg, name, version, revision)
	if err != nil {
		return err
	}

	zap.S().Infow("starting MCP server with STDIO")
	err = mcpserver.ServeStdio(s)
	if err != nil {
		zap.S().Errorw("failed to start STDIO server", "error", err)
		return ierrors.Wrap(err, "failed to start STDIO server")
	}

	zap.S().Infow("STDIO server shutting down")
	return nil
}

func createMCPServer(cfg *config.Config, name string, version string, revision string) (*mcpserver.MCPServer, error) {
	versionString := version
	if revision != "" && revision != "xxx" {
		versionString = versionString + " (" + revision + ")"
	}

	exec := executor.NewExecutor()

	hooks := &mcpserver.Hooks{}
	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		zap.S().Errorw("MCP error occurred",
			"id", id,
			"method", method,
			"error", err,
		)
	})

	zap.S().Debugw("creating MCP server",
		"name", name,
		"version", versionString,
	)
	s := mcpserver.NewMCPServer(
		name,
		versionString,
		mcpserver.WithHooks(hooks),
		mcpserver.WithRecovery(),
	)

	zap.S().Debugw("registering tools")
	if err := RegisterAllTools(s, exec); err != nil {
		zap.S().Errorw("failed to register tools", "error", err)
		return nil, err
	}

	return s, nil
}
