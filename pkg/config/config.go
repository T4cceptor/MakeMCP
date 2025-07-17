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

	"github.com/mark3labs/mcp-go/mcp"
)

// TransportType defines the transport mechanism for the MCP server
type TransportType string

const (
	TransportTypeHTTP  TransportType = "http"
	TransportTypeStdio TransportType = "stdio"
)

// ProcessorStage defines when a processor should run
type ProcessorStage string

const (
	StagePreRequest   ProcessorStage = "pre-request"
	StagePostRequest  ProcessorStage = "post-request"
	StagePreResponse  ProcessorStage = "pre-response"
	StagePostResponse ProcessorStage = "post-response"
)

// ProcessorConfig defines configuration for a processor
type ProcessorConfig struct {
	Name   string                 `json:"name"`
	Stage  ProcessorStage         `json:"stage"`
	Config map[string]interface{} `json:"config"`
}

// SourceConfig defines configuration for a source
type SourceConfig struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// MakeMCPTool extends mcp.Tool with additional MakeMCP information
type MakeMCPTool struct {
	// HandlerInput will be provided to tool handler function as-is
	HandlerInput        map[string]any       `json:"handlerInput,omitempty"`
	OpenAPIHandlerInput *OpenAPIHandlerInput `json:"oapiHandlerInput,omitempty"`

	// HandlerFunction is the actual function that handles the tool call
	HandlerFunction func(
		ctx context.Context,
		request mcp.CallToolRequest,
	) (*mcp.CallToolResult, error) `json:"-"`

	// Processors defines processors applied to each result in order
	Processors []ProcessorConfig `json:"processors,omitempty"`

	// ToolSource holds the original data provided when creating the config
	ToolSource ToolSource `json:"toolSource"`

	mcp.Tool
}

// OpenAPIHandlerInput defines how a particular endpoint is to be called
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

// ToolSource contains source information for a tool
type ToolSource struct {
	URI  string `json:"uri"`  // source of the tool handler data
	Data []byte `json:"data"` // actual input data used to create the MakeMCP config
}

// OpenAPIConfig holds OpenAPI-specific configuration
type OpenAPIConfig struct {
	BaseURL string `json:"baseUrl"`
}

// MakeMCPApp holds all information about the MCP server
type MakeMCPApp struct {
	Name      string        `json:"name"`
	Version   string        `json:"version"`
	Tools     []MakeMCPTool `json:"tools"`
	Transport string        `json:"transport"`
	Port      *string       `json:"port,omitempty"`

	// Source configuration
	Source SourceConfig `json:"source"`

	// Global processors (applied to all tools)
	Processors []ProcessorConfig `json:"processors,omitempty"`

	// Legacy OpenAPI config (for backward compatibility)
	OpenAPIConfig *OpenAPIConfig `json:"openapiConfig,omitempty"`
}

// NewMakeMCPApp creates a new MakeMCPApp with default values
func NewMakeMCPApp(name, version string, transport TransportType) MakeMCPApp {
	return MakeMCPApp{
		Name:      name,
		Version:   version,
		Tools:     []MakeMCPTool{},
		Transport: string(transport),
		Port:      nil,
	}
}

// ToolInputProperty defines a property in the input schema for an MCP tool
type ToolInputProperty struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location"` // OpenAPI 'in' value: path, query, header, cookie, body, etc.
}

// CLIParams holds all CLI parameters for the makemcp commands
type CLIParams struct {
	Specs      string        // URL to OpenAPI specs
	BaseURL    string        // Base URL of the API
	Transport  TransportType // stdio or http
	ConfigOnly bool          // if true, only creates config file
	Port       string        // only valid with transport=http
	DevMode    bool          // if true, disables security warnings
}
