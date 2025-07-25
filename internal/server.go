// Copyright 2025 MakeMCP Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"context"
	"fmt"
	"log"
	"time"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServerFactory abstracts server creation and lifecycle for dependency injection.
type ServerFactory interface {
	CreateHTTPServer(mcpServer *server.MCPServer) HTTPServer
	CreateStdioServer(mcpServer *server.MCPServer) StdioServer
}

// HTTPServer abstracts HTTP server operations.
type HTTPServer interface {
	Start(addr string) error
	Stop() error
}

// StdioServer abstracts stdio server operations.
type StdioServer interface {
	Serve() error
	Stop() error
}

// ProductionServerFactory implements ServerFactory for real server operations.
type ProductionServerFactory struct{}

// CreateHTTPServer creates a production HTTP server wrapper.
func (f *ProductionServerFactory) CreateHTTPServer(mcpServer *server.MCPServer) HTTPServer {
	return &productionHTTPServer{
		server: server.NewStreamableHTTPServer(mcpServer),
	}
}

// CreateStdioServer creates a production stdio server wrapper.
func (f *ProductionServerFactory) CreateStdioServer(mcpServer *server.MCPServer) StdioServer {
	return &productionStdioServer{
		server: mcpServer,
	}
}

// productionHTTPServer wraps the real HTTP server.
type productionHTTPServer struct {
	server *server.StreamableHTTPServer
}

// Start starts the HTTP server on the specified address.
func (s *productionHTTPServer) Start(addr string) error {
	return s.server.Start(addr)
}

// Stop stops the HTTP server.
func (s *productionHTTPServer) Stop() error {
	s.server.Shutdown(context.TODO())
	return nil
}

// productionStdioServer wraps the real stdio server.
type productionStdioServer struct {
	server *server.MCPServer
}

// Serve starts serving the MCP server over stdio.
func (s *productionStdioServer) Serve() error {
	return server.ServeStdio(s.server)
}

// Stop stops the stdio server.
func (s *productionStdioServer) Stop() error {
	// TODO: Implement proper shutdown when mcp-go supports it
	// TODO: check, it might not even be necessary
	return nil
}

// This file provides methods to create and start an MCP server from a MakeMCPApp

// StartServerWithFactory takes a MakeMCPApp and ServerFactory to start an MCP server.
func StartServerWithFactory(app *core.MakeMCPApp, factory ServerFactory) error {
	mcpServer := GetMCPServer(app)

	sharedParams := app.SourceParams.GetSharedParams()
	switch sharedParams.Transport {
	case core.TransportTypeHTTP:
		log.Println("Starting as http MCP server...")
		httpServer := factory.CreateHTTPServer(mcpServer)
		return httpServer.Start(fmt.Sprintf(":%s", sharedParams.Port))

	case core.TransportTypeStdio:
		log.Println("Starting as stdio MCP server...")
		stdioServer := factory.CreateStdioServer(mcpServer)
		if err := stdioServer.Serve(); err != nil {
			log.Printf("Server error: %v\n", err)
			return err
		}
		return nil

	default:
		return fmt.Errorf("unsupported transport type: %s", sharedParams.Transport)
	}
}

// StartServer provides backward compatibility by using the production factory.
func StartServer(app *core.MakeMCPApp) error {
	return StartServerWithFactory(app, &ProductionServerFactory{})
}

// GetMCPServer creates and configures an MCP server from the application configuration..
func GetMCPServer(app *core.MakeMCPApp) *server.MCPServer {
	// Note: app needs to have valid function handlers attached at this point
	mcp_server := server.NewMCPServer(
		app.Name,
		app.Version,
		server.WithToolCapabilities(true),
	)
	for i := range app.Tools {
		tool := (app.Tools[i])
		var mcpTool core.McpTool = tool.ToMcpTool()
		transportAgnosticHandler := tool.GetHandler()

		// Adapt the transport-agnostic handler to work with mcp-go
		mcpGoHandler := adaptHandlerToMcpGo(transportAgnosticHandler)
		mcp_server.AddTool(toMcpGoTool(&mcpTool), mcpGoHandler)
		log.Printf("Registered TOOL: %s with transport-agnostic handler (adapted)", mcpTool.Name)
	}
	return mcp_server
}

func toMcpGoTool(tool *core.McpTool) mcp.Tool {
	return mcp.Tool{
		Name:        tool.Name,
		Description: tool.Description,
		InputSchema: mcp.ToolInputSchema{
			Type:       tool.InputSchema.Type,
			Properties: tool.InputSchema.Properties,
			Required:   tool.InputSchema.Required,
		},
		Annotations: mcp.ToolAnnotation{
			Title:           tool.Annotations.Title,
			ReadOnlyHint:    tool.Annotations.ReadOnlyHint,
			DestructiveHint: tool.Annotations.DestructiveHint,
			IdempotentHint:  tool.Annotations.IdempotentHint,
			OpenWorldHint:   tool.Annotations.OpenWorldHint,
		},
	}
}

/////////////////////////////////////////
// Transport adapters
// - these functions convert between MakeMCP abstraction types and mcp-go types
/////////////////////////////////////////

// mcpRequestToExecutionContext converts an mcp-go request to our abstract execution context.
func mcpRequestToExecutionContext(request mcp.CallToolRequest) core.ToolExecutionContext {
	// Extract parameters from the request arguments
	parameters := request.GetArguments()
	context := core.NewBasicExecutionContext(
		request.Params.Name,
		parameters,
		"", // mcp-go doesn't provide request IDs, we could generate one
	)

	// Set metadata with source information using simplified interface
	context.GetMetadata().Set("mcpMethod", "callTool")
	context.GetMetadata().Set("callTime", time.Now())

	return context
}

// getContentFromString is a helper function that converts any string into a valid mcp.Content map
func getContentFromString(text string) []mcp.Content {
	return []mcp.Content{
		mcp.TextContent{
			Type: "text",
			Text: text,
		},
	}
}

// executionResultToMcpResult converts our abstract execution result to an mcp-go result.
func executionResultToMcpResult(result core.ToolExecutionResult) (*mcp.CallToolResult, error) {
	// Check for error by examining if GetError() returns non-nil
	isError := false
	content := getContentFromString(result.GetContent())
	res := mcp.Result{
		Meta: result.GetMetadata().GetAll(), // Get map from Metadata interface
	}
	if result.GetError() != nil {
		isError = true
		content = getContentFromString(result.GetError().Error())
	}

	return &mcp.CallToolResult{
		Result:  res,
		IsError: isError,
		Content: content,
		// TODO: we only support text results currently
		// - which could be a problem for API returning different content
		// For those cases we likely need some way to pick the right content type
	}, nil
}

// adaptHandlerToMcpGo creates an mcp-go compatible handler from our transport-agnostic handler.
func adaptHandlerToMcpGo(handler core.MakeMcpToolHandler) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Convert mcp-go request to abstract context
		execContext := mcpRequestToExecutionContext(request)

		// Call the transport-agnostic handler
		result, err := handler(ctx, execContext)
		if err != nil {
			// If handler returned an error, create an error result
			return &mcp.CallToolResult{
				IsError: true,
				Content: getContentFromString(err.Error()),
			}, nil
		}

		// Convert abstract result to mcp-go result
		return executionResultToMcpResult(result)
	}
}
