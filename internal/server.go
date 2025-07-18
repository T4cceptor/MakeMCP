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

	"github.com/T4cceptor/MakeMCP/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server" // TODO: dependency on mcp-go
	// TODO: refactor code so only this class interacts with mcp-go
	// this likely requires heavy transformations for types of the different
	// functions
)

// This file provides methods to create and start an MCP server from a MakeMCPApp

// Takes a MakeMCPApp with handler functions and starts an MCP server from it
func StartServer(app *config.MakeMCPApp) error {
	mcpServer := GetMCPServer(app)
	// Start the MCP server
	switch config.TransportType(app.Config.Transport) {
	case config.TransportTypeHTTP:
		log.Println("Starting as http MCP server...")
		streamable_server := server.NewStreamableHTTPServer(mcpServer)
		return streamable_server.Start(
			fmt.Sprintf(":%s", app.Config.Port),
		)
	case config.TransportTypeStdio:
		log.Println("Starting as stdio MCP server...")
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Printf("Server error: %v\n", err)
			return err
		}
	default:
		// TODO: raise error ?!
	}
	return nil
}

func GetMCPServer(app *config.MakeMCPApp) *server.MCPServer {
	mcp_server := server.NewMCPServer(
		app.Name,
		app.Version,
		server.WithToolCapabilities(true),
	)
	for i := range app.Tools {
		tool := &(app.Tools[i])
		mcp_server.AddTool(
			toMcpGoTool(&tool.McpTool),
			tool.HandlerFunction,
		)
		log.Printf("Registered TOOL: %s with func: %p", tool.Name, tool.HandlerFunction)
	}
	return mcp_server
}

func toMcpGoTool(tool *config.McpTool) mcp.Tool {
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

func toMcpGoToolHandler() {
	// TODO: takes a handler function from MakeMCP and return a valid mcp-go handler function
}
