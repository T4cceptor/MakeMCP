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

// This file defines interfaces that allow MakeMCP sources to be completely
// independent of how their tools will be executed. Instead of sources implementing
// handlers that directly use mcp-go (or other framework) types (mcp.CallToolRequest/CallToolResult),
// they work with abstract ToolExecutionContext and ToolExecutionResult interfaces.

// Key benefits of this abstraction:
// 1. Transport Independence: Sources don't know if they're running over MCP stdio,
//    HTTP REST APIs, gRPC, or any other transport mechanism
// 2. Rich Metadata: Results include comprehensive metadata that processors can use
//    to format, filter, redact, or transform outputs appropriately
// 3. Testability: Sources can be tested with mock contexts without mcp-go dependencies
// 4. Future-Proofing: Protocol changes don't break source implementations
// 5. Source Flexibility: Each source can provide its own metadata implementation
//    with only the fields that make sense for that source type
//
// The flow works as follows:
// 1. Transport layer (e.g., internal/server.go) receives a request via MCP, HTTP, etc.
// 2. Transport creates a ToolExecutionContext with the request parameters and metadata
// 3. Source's ToolHandler executes using the abstract context and returns abstract result
// 4. Transport converts the abstract result back to the specific protocol format
// 5. Processors can inspect result metadata to apply appropriate transformations
//
// This design enables sources like OpenAPI to focus purely on their domain logic
// (making HTTP requests) while remaining usable across any transport mechanism.

package core

import (
	"context"
	"time"
)

// ToolExecutionContext provides the input context for tool execution.
// This interface abstracts away transport-specific details and allows sources
// to work with any transport mechanism (MCP, HTTP, gRPC, etc.).
type ToolExecutionContext interface {
	// GetToolName returns the name of the tool being executed
	GetToolName() string

	// GetParameters returns the input parameters for the tool
	GetParameters() map[string]any

	// GetMetadata returns transport-agnostic metadata about the request
	GetMetadata() Metadata
}

// ToolExecutionResult represents the result of tool execution with rich metadata.
// This interface allows processors to understand and transform results appropriately.
type ToolExecutionResult interface {
	// GetContent returns the primary content/output of the tool execution
	GetContent() string

	// GetError returns the error if execution failed, nil otherwise
	GetError() error

	// GetMetadata returns rich metadata about the execution result
	GetMetadata() Metadata
}

// Metadata provides context about the tool execution request.
// Each source type can implement this interface with only the fields that make sense.
type Metadata interface {
	// GetSourceType returns the type of source (e.g., "openapi", "cli", "graphql")
	GetSourceType() string

	// Get returns all metadata key-value pairs
	GetAll() map[string]any

	// Sets the map as metadata
	SetAll(map[string]any)

	// Get returns value associated with provided key
	Get(string) (any, bool)

	// Set stores provided key-value pair internally
	Set(string, any)
}

// MakeMcpToolHandler is the transport-agnostic function signature for tool execution.
// Sources implement handlers with this signature, making them usable with any transport.
type MakeMcpToolHandler func(ctx context.Context, request ToolExecutionContext) (ToolExecutionResult, error)

// =============================================================================
// Basic Implementations
// =============================================================================
//
// The following provides simple, practical implementations of the above interfaces
// that focus on usability while remaining extensible.

// BasicExecutionContext is a simple implementation of ToolExecutionContext.
type BasicExecutionContext struct {
	toolName   string
	parameters map[string]any
	metadata   Metadata
}

// NewBasicExecutionContext creates a new BasicExecutionContext.
func NewBasicExecutionContext(toolName string, parameters map[string]any, requestID string) *BasicExecutionContext {
	return &BasicExecutionContext{
		toolName:   toolName,
		parameters: parameters,
		metadata:   NewBasicMetadata(""),
	}
}

// GetToolName returns the name of the tool being executed.
func (b *BasicExecutionContext) GetToolName() string { return b.toolName }

// GetParameters returns the input parameters for the tool.
func (b *BasicExecutionContext) GetParameters() map[string]any { return b.parameters }

// GetMetadata returns the execution metadata.
func (b *BasicExecutionContext) GetMetadata() Metadata { return b.metadata }

// BasicExecutionResult is a simple implementation of ToolExecutionResult.
// Uses the Metadata interface for consistency with execution context.
type BasicExecutionResult struct {
	content  string
	err      error
	metadata Metadata
}

// NewBasicExecutionResult creates a successful execution result.
func NewBasicExecutionResult(content string, err error) *BasicExecutionResult {
	metadata := NewBasicMetadata("")
	if err != nil {
		content += err.Error()
		metadata.Set("isError", true)
		metadata.Set("error", err.Error())
	}
	return &BasicExecutionResult{
		content:  content,
		err:      err,
		metadata: metadata,
	}
}

// NewBasicExecutionError creates an error execution result.
func NewBasicExecutionError(err error) *BasicExecutionResult {
	content := ""
	if err != nil {
		content = err.Error()
	}

	return &BasicExecutionResult{
		content:  content,
		err:      err,
		metadata: NewBasicMetadata(""),
	}
}

// GetContent returns the primary content/output of the tool execution.
func (b *BasicExecutionResult) GetContent() string { return b.content }

// GetError returns the error if execution failed, nil otherwise.
func (b *BasicExecutionResult) GetError() error { return b.err }

// GetMetadata returns the execution metadata.
func (b *BasicExecutionResult) GetMetadata() Metadata { return b.metadata }

// BasicExecutionMetadata is a simple implementation of Metadata.
type BasicExecutionMetadata struct {
	sourceType string
	data       map[string]any
}

// NewBasicMetadata creates a new BasicExecutionMetadata with consistent naming.
func NewBasicMetadata(sourceType string) *BasicExecutionMetadata {
	return &BasicExecutionMetadata{
		sourceType: sourceType,
		data:       make(map[string]any),
	}
}

// NewBasicExecutionMetadata creates a new BasicExecutionMetadata with request time (deprecated, use NewBasicMetadata).
func NewBasicExecutionMetadata(sourceType string, requestTime time.Time) *BasicExecutionMetadata {
	metadata := &BasicExecutionMetadata{
		sourceType: sourceType,
		data:       make(map[string]any),
	}
	metadata.data["requestTime"] = requestTime
	return metadata
}

// GetSourceType returns the type of source (e.g., "openapi", "cli", "graphql").
func (b *BasicExecutionMetadata) GetSourceType() string { return b.sourceType }

// GetAll returns all metadata key-value pairs.
func (b *BasicExecutionMetadata) GetAll() map[string]any { return b.data }

// SetAll replaces all metadata with the provided key-value pairs.
func (b *BasicExecutionMetadata) SetAll(data map[string]any) { b.data = data }

// Get returns the value associated with the provided key.
func (b *BasicExecutionMetadata) Get(key string) (any, bool) {
	val, ok := b.data[key]
	return val, ok
}

// Set stores the provided key-value pair in the metadata.
func (b *BasicExecutionMetadata) Set(key string, value any) {
	b.data[key] = value
}
