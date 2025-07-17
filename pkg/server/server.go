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

package server

import (
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/server"

	"github.com/T4cceptor/MakeMCP/pkg/config"
	"github.com/T4cceptor/MakeMCP/pkg/tool"
)

// MCPServer wraps the mcp-go server with additional functionality
type MCPServer struct {
	server *server.MCPServer
	app    *config.MakeMCPApp
}

// NewMCPServer creates a new MCP server from configuration
func NewMCPServer(app *config.MakeMCPApp) *MCPServer {
	mcpServer := server.NewMCPServer(
		app.Name,
		app.Version,
		server.WithToolCapabilities(true),
	)

	// Create the wrapper
	s := &MCPServer{
		server: mcpServer,
		app:    app,
	}

	// Register tools
	s.registerTools()

	return s
}

// registerTools registers all tools from the app configuration
func (s *MCPServer) registerTools() {
	for i := range s.app.Tools {
		tool := &s.app.Tools[i]
		
		// Create tool handler with processor chain
		handler := s.createToolHandler(tool)
		
		// Register with MCP server
		s.server.AddTool(tool.Tool, handler)
		log.Printf("Registered TOOL: %s with handler", tool.Name)
	}
}

// createToolHandler creates a handler function for a tool with processor chain
func (s *MCPServer) createToolHandler(mcpTool *config.MakeMCPTool) tool.HandlerFunc {
	return tool.NewHandler(mcpTool, s.app.Processors)
}

// Start starts the MCP server with the specified transport
func (s *MCPServer) Start(transport config.TransportType, port string) error {
	switch transport {
	case config.TransportTypeHTTP:
		log.Printf("Starting HTTP MCP server on port %s...", port)
		streamableServer := server.NewStreamableHTTPServer(s.server)
		return streamableServer.Start(fmt.Sprintf(":%s", port))
	case config.TransportTypeStdio:
		log.Println("Starting stdio MCP server...")
		return server.ServeStdio(s.server)
	default:
		return fmt.Errorf("unsupported transport type: %s", transport)
	}
}

// GetServer returns the underlying mcp-go server
func (s *MCPServer) GetServer() *server.MCPServer {
	return s.server
}

// GetApp returns the app configuration
func (s *MCPServer) GetApp() *config.MakeMCPApp {
	return s.app
}

// AddTool adds a tool to the server
func (s *MCPServer) AddTool(mcpTool *config.MakeMCPTool) {
	handler := s.createToolHandler(mcpTool)
	s.server.AddTool(mcpTool.Tool, handler)
	s.app.Tools = append(s.app.Tools, *mcpTool)
	log.Printf("Added TOOL: %s", mcpTool.Name)
}

// RemoveTool removes a tool from the server (note: mcp-go doesn't support removal)
func (s *MCPServer) RemoveTool(toolName string) error {
	// Note: The underlying mcp-go server doesn't support removing tools
	// This would need to be implemented if the library supports it in the future
	return fmt.Errorf("tool removal not supported by underlying MCP server")
}