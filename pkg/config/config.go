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

// IsValid returns true if the transport type is valid
func (t TransportType) IsValid() bool {
	switch t {
	case TransportTypeHTTP, TransportTypeStdio:
		return true
	default:
		return false
	}
}

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

type MakeMCPTool interface {
	GetName() string
	GetHandler() func(
		ctx context.Context,
		request mcp.CallToolRequest,
		// TODO: refactor to get rid of mcp-go dependency
	) (*mcp.CallToolResult, error)
	ToMcpTool() McpTool
	ToJSON() string
}

// MakeMCPApp holds all information about the MCP server
// Main data structure
type MakeMCPApp struct {
	Name       string        `json:"name"`       // Name of the App
	Version    string        `json:"version"`    // Version of the app
	SourceType string        `json:"sourceType"` // Type of source (openapi, cli, etc.)
	Tools      []MakeMCPTool `json:"tools"`      // Tools the MCP server will provide
	CliParams  CLIParams     `json:"config"`
}

// NewMakeMCPApp creates a new MakeMCPApp with default values
func NewMakeMCPApp(name, version string, transport TransportType) MakeMCPApp {
	// Validate transport type - if invalid, default to stdio
	if !transport.IsValid() {
		transport = TransportTypeStdio
	}

	return MakeMCPApp{
		Name:       name,
		Version:    version,
		SourceType: "", // Will be set by source during parsing
		Tools:      []MakeMCPTool{},
		CliParams: CLIParams{
			BaseCLIParams: BaseCLIParams{
				Transport: transport,
			},
		},
	}
}

// BaseCLIParams holds all CLI parameters for the makemcp commands
type BaseCLIParams struct {
	// Generic MCP server parameters
	Transport  TransportType `json:"transport"`  // stdio or http
	ConfigOnly bool          `json:"configOnly"` // if true, only creates config file
	Port       string        `json:"port"`       // only valid with transport=http
	DevMode    bool          `json:"devMode"`    // true if running in development mode - related to security checks
	SourceType string        `json:"type"`       // type of source (openapi, cli, etc.)
}

// CLIParams holds all CLI parameters for the makemcp commands
type CLIParams struct {
	// Generic MCP server parameters
	BaseCLIParams

	// Source-specific parameters
	CliFlags map[string]any `json:"flags"` // source-specific configuration
	CliArgs  []string       `json:"args"`
}

func (c *CLIParams) GetFlag(flag string) any {
	return c.CliFlags[flag]
}

// ToJSON returns a JSON representation of the CLIParams for logging and debugging
func (c CLIParams) ToJSON() string {
	jsonBytes, err := json.Marshal(c)
	if err != nil {
		return `{"error": "failed to marshal CLIParams to JSON"}`
	}
	return string(jsonBytes)
}
