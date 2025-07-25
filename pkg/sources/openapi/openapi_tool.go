package openapi

import (
	core "github.com/T4cceptor/MakeMCP/pkg/core"
)

// OpenAPIMcpTool represents an MCP tool generated from an OpenAPI operation.
// This tool is now transport-agnostic and can work with any transport mechanism.
type OpenAPIMcpTool struct {
	core.McpTool
	OpenAPIHandlerInput *OpenAPIHandlerInput    `json:"oapiHandlerInput,omitempty"`
	handler             core.MakeMcpToolHandler `json:"-"`
}

// GetName returns the name of the OpenAPI MCP tool.
func (o *OpenAPIMcpTool) GetName() string {
	return o.McpTool.Name
}

// ToMcpTool returns the core MCP tool representation.
func (o *OpenAPIMcpTool) ToMcpTool() core.McpTool {
	return o.McpTool
}

// GetHandler returns the transport-agnostic tool handler function for processing requests.
func (o *OpenAPIMcpTool) GetHandler() core.MakeMcpToolHandler {
	return o.handler
}
