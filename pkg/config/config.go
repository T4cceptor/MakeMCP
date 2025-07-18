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

package config

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// TransportType defines the transport mechanism for the MCP server
type TransportType string

const (
	TransportTypeHTTP  TransportType = "http"
	TransportTypeStdio TransportType = "stdio"
)

// MCP objects
type McpToolInputSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

type McpToolAnnotation struct {
	// Human-readable title for the tool
	Title string `json:"title,omitempty"`
	// If true, the tool does not modify its environment
	ReadOnlyHint *bool `json:"readOnlyHint,omitempty"`
	// If true, the tool may perform destructive updates
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	// If true, repeated calls with same args have no additional effect
	IdempotentHint *bool `json:"idempotentHint,omitempty"`
	// If true, tool interacts with external entities
	OpenWorldHint *bool `json:"openWorldHint,omitempty"`
}

type McpTool struct {
	// The name of the tool.
	Name string `json:"name"`
	// A human-readable description of the tool.
	Description string `json:"description,omitempty"`
	// A JSON Schema object defining the expected parameters for the tool.
	InputSchema McpToolInputSchema `json:"inputSchema"`
	// Alternative to InputSchema - allows arbitrary JSON Schema to be provided
	RawInputSchema json.RawMessage `json:"-"` // Hide this from JSON marshaling
	// Optional properties describing tool behavior
	Annotations McpToolAnnotation `json:"annotations"`
}

// MakeMCPTool extends McpTool with additional MakeMCP information
type MakeMCPTool struct {
	// HandlerInput will be provided to tool handler function as-is
	HandlerInput        map[string]any       `json:"handlerInput,omitempty"`
	OpenAPIHandlerInput *OpenAPIHandlerInput `json:"oapiHandlerInput,omitempty"`

	// HandlerFunction is the actual function that handles the tool call
	HandlerFunction func(
		ctx context.Context,
		request mcp.CallToolRequest,
		// TODO: refactor to get rid of mcp dependency
	) (*mcp.CallToolResult, error) `json:"-"`

	McpTool
}

// OpenAPIHandlerInput defines how a particular endpoint is to be called
// TODO: move into OpenAPI source folder
type OpenAPIHandlerInput struct {
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Headers    map[string]string `json:"headers"`
	Cookies    map[string]string `json:"cookies"`
	BodyAppend map[string]any    `json:"bodyAppend"`
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

// MakeMCPApp holds all information about the MCP server
// Main data structure
type MakeMCPApp struct {
	Name    string        `json:"name"`    // Name of the App
	Version string        `json:"version"` // Version of the app
	Tools   []MakeMCPTool `json:"tools"`   // Tools the MCP server will provide
	Config  Config        `json:"config"`
}

// NewMakeMCPApp creates a new MakeMCPApp with default values
func NewMakeMCPApp(name, version string, transport TransportType) MakeMCPApp {
	return MakeMCPApp{
		Name:    name,
		Version: version,
		Tools:   []MakeMCPTool{},
		Config: Config{
			Transport: transport,
		},
	}
}

// Config holds all CLI parameters for the makemcp commands
type Config struct {
	// Generic MCP server parameters
	Transport  TransportType `json:"transport"`  // stdio or http
	ConfigOnly bool          `json:"configOnly"` // if true, only creates config file
	Port       string        `json:"port"`       // only valid with transport=http
	DevMode    bool          `json:"devMode"`    // true if running in development mode - related to security checks

	// Source-specific parameters
	SourceType string         `json:"type"`  // type of source (openapi, cli, etc.)
	CliFlags   map[string]any `json:"flags"` // source-specific configuration
	CliArgs    []string       `json:"args"`
}

// ToJSON returns a JSON representation of the CLIParams for logging and debugging
func (c Config) ToJSON() string {
	jsonBytes, err := json.Marshal(c)
	if err != nil {
		return `{"error": "failed to marshal CLIParams to JSON"}`
	}
	return string(jsonBytes)
}
