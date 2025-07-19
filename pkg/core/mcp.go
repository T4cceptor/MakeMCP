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

package core

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

// MCP protocol types

// McpToolInputSchema defines the JSON Schema for tool input parameters
type McpToolInputSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

// McpToolAnnotation provides metadata about tool behavior and characteristics
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

// McpTool represents an MCP tool definition
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

// MakeMCPTool defines the interface that all MCP tools must implement
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