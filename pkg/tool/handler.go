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

package tool

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/T4cceptor/MakeMCP/pkg/config"
	"github.com/T4cceptor/MakeMCP/pkg/processors"
)

// HandlerFunc is the type for tool handler functions
type HandlerFunc = server.ToolHandlerFunc

// NewHandler creates a new tool handler with processor chain support
func NewHandler(tool *config.MakeMCPTool, globalProcessors []config.ProcessorConfig) HandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Combine global and tool-specific processors
		allProcessors := append(globalProcessors, tool.Processors...)
		
		// Create processor data
		data := &processors.ProcessorData{
			Request:  &request,
			Tool:     tool,
			Metadata: make(map[string]interface{}),
		}

		// Pre-request processing
		if err := processors.ProcessWithChain(ctx, allProcessors, config.StagePreRequest, data); err != nil {
			return nil, fmt.Errorf("pre-request processing failed: %w", err)
		}

		// Execute the actual tool
		result, err := executeToolHandler(ctx, tool, request, data)
		if err != nil {
			return nil, err
		}

		// Set result in processor data
		data.Result = result

		// Post-response processing
		if err := processors.ProcessWithChain(ctx, allProcessors, config.StagePostResponse, data); err != nil {
			return nil, fmt.Errorf("post-response processing failed: %w", err)
		}

		return data.Result, nil
	}
}

// executeToolHandler executes the base tool handler based on the tool type
func executeToolHandler(ctx context.Context, tool *config.MakeMCPTool, request mcp.CallToolRequest, data *processors.ProcessorData) (*mcp.CallToolResult, error) {
	// If the tool has a custom handler function, use it
	if tool.HandlerFunction != nil {
		return tool.HandlerFunction(ctx, request)
	}

	// Handle OpenAPI tools
	if tool.OpenAPIHandlerInput != nil {
		return handleOpenAPITool(ctx, tool, request, data)
	}

	// Default response for unknown tool types
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Tool %s executed successfully", tool.Name),
			},
		},
		IsError: false,
	}, nil
}

// handleOpenAPITool handles OpenAPI-based tools
func handleOpenAPITool(ctx context.Context, tool *config.MakeMCPTool, request mcp.CallToolRequest, data *processors.ProcessorData) (*mcp.CallToolResult, error) {
	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Extract parameters from the request
	// 2. Build an HTTP request based on OpenAPIHandlerInput
	// 3. Execute the HTTP request
	// 4. Process the response
	// 5. Return the result

	// For now, return a simple response
	content := fmt.Sprintf("OpenAPI tool %s (%s %s) would be executed with arguments: %v", 
		tool.Name, 
		tool.OpenAPIHandlerInput.Method, 
		tool.OpenAPIHandlerInput.Path, 
		request.Params.Arguments)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: content,
			},
		},
		IsError: false,
	}, nil
}

// createHTTPRequest creates an HTTP request from OpenAPI handler input
func createHTTPRequest(ctx context.Context, tool *config.MakeMCPTool, request mcp.CallToolRequest) (*http.Request, error) {
	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Parse the request arguments
	// 2. Split parameters by location (path, query, header, body)
	// 3. Build the URL with path parameters
	// 4. Add query parameters
	// 5. Set headers and cookies
	// 6. Create request body
	// 7. Return the HTTP request

	// For now, return a simple GET request
	req, err := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	return req, nil
}