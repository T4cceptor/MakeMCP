package openapi

import (
	"context"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
	"github.com/mark3labs/mcp-go/mcp"
)

// OpenAPIMcpTool represents an MCP tool generated from an OpenAPI operation.
type OpenAPIMcpTool struct {
	core.McpTool
	OpenAPIHandlerInput *OpenAPIHandlerInput `json:"oapiHandlerInput,omitempty"`
	handler             func(
		ctx context.Context,
		request mcp.CallToolRequest,
		// TODO: refactor to get rid of mcp-go dependency
	) (*mcp.CallToolResult, error) `json:"-"`
}

// GetName returns the name of the OpenAPI MCP tool.
func (o *OpenAPIMcpTool) GetName() string {
	return o.McpTool.Name
}

// ToMcpTool returns the core MCP tool representation.
func (o *OpenAPIMcpTool) ToMcpTool() core.McpTool {
	return o.McpTool
}

// GetHandler returns the MCP tool handler function for processing requests.
func (o *OpenAPIMcpTool) GetHandler() func(
	ctx context.Context,
	request mcp.CallToolRequest,
	// TODO: refactor to get rid of mcp-go dependency
) (*mcp.CallToolResult, error) {
	return o.handler
}
