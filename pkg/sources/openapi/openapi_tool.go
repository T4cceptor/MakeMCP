package openapi

import (
	core "github.com/T4cceptor/MakeMCP/pkg/core"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// OpenAPIMcpTool represents an MCP tool generated from an OpenAPI operation.
// This tool is now transport-agnostic and can work with any transport mechanism.
type OpenAPIMcpTool struct {
	core.McpTool
	OpenAPIHandlerInput *OpenAPIHandlerInput    `json:"oapiHandlerInput,omitempty"`
	handler             core.MakeMcpToolHandler `json:"-"`
	Operation           *v3.Operation           `json:"-"`
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

// OpenAPIHandlerInput defines how a particular endpoint is to be called
type OpenAPIHandlerInput struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
	Cookies     map[string]string `json:"cookies"`
	BodyAppend  map[string]any    `json:"bodyAppend"`
	ContentType string            `json:"contentType,omitempty"`
}

// NewOpenAPIHandlerInput creates a new OpenAPIHandlerInput
func NewOpenAPIHandlerInput(method, path string) OpenAPIHandlerInput {
	return OpenAPIHandlerInput{
		Method:     method,
		Path:       path,
		Headers:    make(map[string]string),
		Cookies:    make(map[string]string),
		BodyAppend: make(map[string]any),
	}
}
