package main

import (
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type TransportType string

const (
	TransportTypeHTTP  TransportType = "http"
	TransportTypeStdio TransportType = "stdio"
)

// APIClient struct to encapsulate baseURL and http.Client
type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// Internal struct to hold config data
// TODO: get rid of this in favor of other data structures
type MCPToolConfig struct {
	Name        string
	Description string
	Options     []mcp.ToolOption
	Handler     server.ToolHandlerFunc
}

// MCPResource represents a resource with optional metadata fields.
type MCPResource struct {
	URI         string  `json:"uri"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	MimeType    *string `json:"mimeType,omitempty"`
	Size        *int64  `json:"size,omitempty"`
}

// MCPResourceTemplate represents a template for resources, with optional metadata fields.
type MCPResourceTemplate struct {
	URITemplate string  `json:"uriTemplate"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	MimeType    *string `json:"mimeType,omitempty"`
}

// ToolAnnotation represents optional hints about tool behavior.
type ToolAnnotation struct {
	Title           *string `json:"title,omitempty"`
	ReadOnlyHint    *bool   `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool   `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool   `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool   `json:"openWorldHint,omitempty"`
}

type InputSchema struct {
	Type       string         `json:"type"`               // should be "object" by default, can be of a pre-defined type, in this case the object is mapped against this type
	Properties map[string]any `json:"inputSchema"`        // defines the properties this tool has
	Required   []string       `json:"required,omitempty"` // defines which properties are required
}

// MCPTool represents a tool definition with input schema and optional annotations.
type MCPTool struct {
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	InputSchema InputSchema     `json:"inputSchema"`
	Annotations *ToolAnnotation `json:"annotations,omitempty"`
}

// PromptArgument represents an argument for a prompt.
type PromptArgument struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Required    *bool   `json:"required,omitempty"`
}

// MCPPrompt represents a prompt definition with optional arguments.
type MCPPrompt struct {
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// MCPConfig holds all resources, resource templates, and tools for the configuration.
type MCPConfig struct {
	Resources         []MCPResource         `json:"resources"`
	ResourceTemplates []MCPResourceTemplate `json:"resourceTemplates"`
	Tools             []MCPTool             `json:"tools"`
	Prompts           []MCPPrompt           `json:"prompts"`
}

/*
OpenAPI:
- base url
- endpoints
	-> endpoints have to provide all necessary information to actually call the base url + path
	-> method + path + params
	-> params (incl. body, headers, query params, and path params)
	have to be provided by the AI when calling the endpoint

*/

type MakeMCPProcessor struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
	// TODO: check if this is appropriate or if we should use a list instead
}

// Data provided to create the corresponding operation in MakeMCP
type MakeMCPInput struct {
	URI  string         `json:"name"`
	Data map[string]any `json:"data"`
}

// Additional MakeMCP information for a Tool.
// The tool is defined by its name and the corresponding namespace.
type MakeMCPTool struct {
	Name         string             `json:"name"`
	HandlerInput map[string]any     `json:"handlerInput,omitempty"` // will be provided to tool handler function as-is, the tool handler needs to unmarshall this properly
	Processors   []MakeMCPProcessor `json:"processors,omitempty"`
	MakeMCPInput MakeMCPInput       `json:"makeMCPInput"` // holds thze original data provided when creating the config
}

type MakeMCPConfig struct {
	Tools map[string]MakeMCPTool `json:"tools"` // we only support Tools for now
	// TODO: add other fields as well: resources, resourceTemplates, prompts
}

type MakeMCPNamespace struct {
	Mcp    MCPConfig     `json:"mcp"`
	Config MakeMCPConfig `json:"config"`
}

// TODO: check if this is correct -> how to define the root of a JSON object ?
type RootConfig struct {
	Namespaces map[string]MakeMCPNamespace `json:""`
}
