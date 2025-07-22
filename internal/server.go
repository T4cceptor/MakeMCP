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
	"fmt"
	"log"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server" // TODO: dependency on mcp-go
	// TODO: refactor code so only this class interacts with mcp-go.
	// this likely requires heavy transformations for types of the different
	// functions
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
		mcpServer: mcpServer,
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
	// TODO: Implement proper shutdown when mcp-go supports it
	return nil
}

// productionStdioServer wraps the real stdio server.
type productionStdioServer struct {
	mcpServer *server.MCPServer
}

// Serve starts serving the MCP server over stdio.
func (s *productionStdioServer) Serve() error {
	return server.ServeStdio(s.mcpServer)
}

// Stop stops the stdio server.
func (s *productionStdioServer) Stop() error {
	// TODO: Implement proper shutdown when mcp-go supports it
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
	mcp_server := server.NewMCPServer(
		app.Name,
		app.Version,
		server.WithToolCapabilities(true),
	)
	for i := range app.Tools {
		tool := (app.Tools[i])
		var mcpTool core.McpTool = tool.ToMcpTool()
		handlerFunc := tool.GetHandler()
		mcp_server.AddTool(
			toMcpGoTool(&mcpTool),
			handlerFunc,
		)
		log.Printf("Registered TOOL: %s with func: %p", mcpTool.Name, handlerFunc)
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
