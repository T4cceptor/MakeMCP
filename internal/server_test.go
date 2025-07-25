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

package internal

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
	"github.com/T4cceptor/MakeMCP/pkg/sources/openapi"
	"github.com/mark3labs/mcp-go/server"
)

func TestToMcpGoTool(t *testing.T) {
	// Test basic tool conversion
	t.Run("Basic tool conversion", func(t *testing.T) {
		tool := &core.McpTool{
			Name:        "test_tool",
			Description: "A test tool",
			InputSchema: core.McpToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"param1": map[string]any{
						"type":        "string",
						"description": "First parameter",
					},
					"param2": map[string]any{
						"type":        "integer",
						"description": "Second parameter",
					},
				},
				Required: []string{"param1"},
			},
			Annotations: core.McpToolAnnotation{
				Title:           "Test Tool",
				ReadOnlyHint:    boolPtr(true),
				DestructiveHint: boolPtr(false),
				IdempotentHint:  boolPtr(true),
				OpenWorldHint:   boolPtr(false),
			},
		}

		mcpTool := toMcpGoTool(tool)

		// Verify basic fields
		if mcpTool.Name != tool.Name {
			t.Errorf("Expected name '%s', got '%s'", tool.Name, mcpTool.Name)
		}

		if mcpTool.Description != tool.Description {
			t.Errorf("Expected description '%s', got '%s'", tool.Description, mcpTool.Description)
		}

		// Verify input schema
		if mcpTool.InputSchema.Type != tool.InputSchema.Type {
			t.Errorf("Expected schema type '%s', got '%s'", tool.InputSchema.Type, mcpTool.InputSchema.Type)
		}

		if !reflect.DeepEqual(mcpTool.InputSchema.Properties, tool.InputSchema.Properties) {
			t.Errorf("Properties don't match. Expected %v, got %v", tool.InputSchema.Properties, mcpTool.InputSchema.Properties)
		}

		if !reflect.DeepEqual(mcpTool.InputSchema.Required, tool.InputSchema.Required) {
			t.Errorf("Required fields don't match. Expected %v, got %v", tool.InputSchema.Required, mcpTool.InputSchema.Required)
		}

		// Verify annotations
		if mcpTool.Annotations.Title != tool.Annotations.Title {
			t.Errorf("Expected title '%s', got '%s'", tool.Annotations.Title, mcpTool.Annotations.Title)
		}

		if !reflect.DeepEqual(mcpTool.Annotations.ReadOnlyHint, tool.Annotations.ReadOnlyHint) {
			t.Errorf("ReadOnlyHint doesn't match. Expected %v, got %v", tool.Annotations.ReadOnlyHint, mcpTool.Annotations.ReadOnlyHint)
		}

		if !reflect.DeepEqual(mcpTool.Annotations.DestructiveHint, tool.Annotations.DestructiveHint) {
			t.Errorf("DestructiveHint doesn't match. Expected %v, got %v", tool.Annotations.DestructiveHint, mcpTool.Annotations.DestructiveHint)
		}

		if !reflect.DeepEqual(mcpTool.Annotations.IdempotentHint, tool.Annotations.IdempotentHint) {
			t.Errorf("IdempotentHint doesn't match. Expected %v, got %v", tool.Annotations.IdempotentHint, mcpTool.Annotations.IdempotentHint)
		}

		if !reflect.DeepEqual(mcpTool.Annotations.OpenWorldHint, tool.Annotations.OpenWorldHint) {
			t.Errorf("OpenWorldHint doesn't match. Expected %v, got %v", tool.Annotations.OpenWorldHint, mcpTool.Annotations.OpenWorldHint)
		}
	})

	// Test tool with minimal fields
	t.Run("Minimal tool conversion", func(t *testing.T) {
		tool := &core.McpTool{
			Name:        "minimal_tool",
			Description: "",
			InputSchema: core.McpToolInputSchema{
				Type:       "object",
				Properties: nil,
				Required:   nil,
			},
			Annotations: core.McpToolAnnotation{},
		}

		mcpTool := toMcpGoTool(tool)

		if mcpTool.Name != "minimal_tool" {
			t.Errorf("Expected name 'minimal_tool', got '%s'", mcpTool.Name)
		}

		if mcpTool.Description != "" {
			t.Errorf("Expected empty description, got '%s'", mcpTool.Description)
		}

		if mcpTool.InputSchema.Type != "object" {
			t.Errorf("Expected schema type 'object', got '%s'", mcpTool.InputSchema.Type)
		}

		if mcpTool.InputSchema.Properties != nil {
			t.Errorf("Expected nil properties, got %v", mcpTool.InputSchema.Properties)
		}

		if mcpTool.InputSchema.Required != nil {
			t.Errorf("Expected nil required, got %v", mcpTool.InputSchema.Required)
		}
	})

	// Test tool with empty properties map
	t.Run("Tool with empty properties", func(t *testing.T) {
		tool := &core.McpTool{
			Name:        "empty_props_tool",
			Description: "Tool with empty properties",
			InputSchema: core.McpToolInputSchema{
				Type:       "object",
				Properties: make(map[string]any),
				Required:   []string{},
			},
			Annotations: core.McpToolAnnotation{
				Title: "Empty Props Tool",
			},
		}

		mcpTool := toMcpGoTool(tool)

		if len(mcpTool.InputSchema.Properties) != 0 {
			t.Errorf("Expected empty properties map, got %v", mcpTool.InputSchema.Properties)
		}

		if len(mcpTool.InputSchema.Required) != 0 {
			t.Errorf("Expected empty required slice, got %v", mcpTool.InputSchema.Required)
		}
	})
}

func TestGetMCPServer(t *testing.T) {
	// Create a test app with mock tools
	app := &core.MakeMCPApp{
		Name:    "Test Server",
		Version: "1.0.0",
		Tools:   []core.MakeMCPTool{&mockMakeMCPTool{}},
	}

	server := GetMCPServer(app)

	// Test that server is created
	if server == nil {
		t.Fatal("Expected non-nil MCP server")
	}

	// Note: We can't easily test the internal state of the MCP server
	// without exposing private fields, but we can verify it doesn't panic
	// and returns a valid server instance.
}

func TestStartServerWithFactory(t *testing.T) {
	t.Run("HTTP transport success", func(t *testing.T) {
		app := &core.MakeMCPApp{
			Name:    "HTTP Test Server",
			Version: "1.0.0",
			SourceParams: &openapi.OpenAPIParams{
				BaseAppParams: &core.BaseAppParams{
					Transport:  core.TransportTypeHTTP,
					Port:       "8080",
					SourceType: "openapi",
				},
				Specs:   "http://example.com/openapi.json",
				BaseURL: "http://example.com",
				Timeout: 30,
			},
			Tools: []core.MakeMCPTool{},
		}

		mockFactory := &MockServerFactory{}
		err := StartServerWithFactory(app, mockFactory)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !mockFactory.HTTPStartCalled {
			t.Error("Expected HTTP server Start() to be called")
		}

		if mockFactory.StdioServeCalled {
			t.Error("Expected stdio server Serve() NOT to be called")
		}
	})

	t.Run("HTTP transport error", func(t *testing.T) {
		app := &core.MakeMCPApp{
			Name:    "HTTP Test Server",
			Version: "1.0.0",
			SourceParams: &openapi.OpenAPIParams{
				BaseAppParams: &core.BaseAppParams{
					Transport:  core.TransportTypeHTTP,
					Port:       "8080",
					SourceType: "openapi",
				},
				Specs:   "http://example.com/openapi.json",
				BaseURL: "http://example.com",
				Timeout: 30,
			},
			Tools: []core.MakeMCPTool{},
		}

		expectedError := fmt.Errorf("failed to bind port")
		mockFactory := &MockServerFactory{
			HTTPStartError: expectedError,
		}

		err := StartServerWithFactory(app, mockFactory)

		if err != expectedError {
			t.Errorf("Expected error %v, got: %v", expectedError, err)
		}

		if !mockFactory.HTTPStartCalled {
			t.Error("Expected HTTP server Start() to be called even on error")
		}
	})

	t.Run("Stdio transport success", func(t *testing.T) {
		app := &core.MakeMCPApp{
			Name:    "Stdio Test Server",
			Version: "1.0.0",
			SourceParams: &openapi.OpenAPIParams{
				BaseAppParams: &core.BaseAppParams{
					Transport:  core.TransportTypeStdio,
					Port:       "8080", // Should be ignored for stdio
					SourceType: "openapi",
				},
				Specs:   "http://example.com/openapi.json",
				BaseURL: "http://example.com",
				Timeout: 30,
			},
			Tools: []core.MakeMCPTool{},
		}

		mockFactory := &MockServerFactory{}
		err := StartServerWithFactory(app, mockFactory)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !mockFactory.StdioServeCalled {
			t.Error("Expected stdio server Serve() to be called")
		}

		if mockFactory.HTTPStartCalled {
			t.Error("Expected HTTP server Start() NOT to be called")
		}
	})

	t.Run("Stdio transport error", func(t *testing.T) {
		app := &core.MakeMCPApp{
			Name:    "Stdio Test Server",
			Version: "1.0.0",
			SourceParams: &openapi.OpenAPIParams{
				BaseAppParams: &core.BaseAppParams{
					Transport:  core.TransportTypeStdio,
					SourceType: "openapi",
				},
				Specs:   "http://example.com/openapi.json",
				BaseURL: "http://example.com",
				Timeout: 30,
			},
			Tools: []core.MakeMCPTool{},
		}

		expectedError := fmt.Errorf("stdio server failed")
		mockFactory := &MockServerFactory{
			StdioServeError: expectedError,
		}

		err := StartServerWithFactory(app, mockFactory)

		if err != expectedError {
			t.Errorf("Expected error %v, got: %v", expectedError, err)
		}

		if !mockFactory.StdioServeCalled {
			t.Error("Expected stdio server Serve() to be called even on error")
		}
	})

	t.Run("Invalid transport type", func(t *testing.T) {
		app := &core.MakeMCPApp{
			Name:    "Invalid Transport Server",
			Version: "1.0.0",
			SourceParams: &openapi.OpenAPIParams{
				BaseAppParams: &core.BaseAppParams{
					Transport:  "invalid",
					Port:       "8080",
					SourceType: "openapi",
				},
				Specs:   "http://example.com/openapi.json",
				BaseURL: "http://example.com",
				Timeout: 30,
			},
			Tools: []core.MakeMCPTool{},
		}

		mockFactory := &MockServerFactory{}
		err := StartServerWithFactory(app, mockFactory)

		if err == nil {
			t.Error("Expected error for invalid transport")
		}

		if !strings.Contains(err.Error(), "unsupported transport type") {
			t.Errorf("Expected unsupported transport error, got: %v", err)
		}

		if mockFactory.HTTPStartCalled || mockFactory.StdioServeCalled {
			t.Error("Expected no server methods to be called for invalid transport")
		}
	})
}

func TestStartServer(t *testing.T) {
	// Test backward compatibility - StartServer should use ProductionServerFactory
	t.Run("Backward compatibility with invalid transport", func(t *testing.T) {
		app := &core.MakeMCPApp{
			Name:    "Test Server",
			Version: "1.0.0",
			SourceParams: &openapi.OpenAPIParams{
				BaseAppParams: &core.BaseAppParams{
					Transport:  "invalid",
					Port:       "8080",
					SourceType: "openapi",
				},
				Specs:   "http://example.com/openapi.json",
				BaseURL: "http://example.com",
				Timeout: 30,
			},
			Tools: []core.MakeMCPTool{},
		}

		err := StartServer(app)

		if err == nil {
			t.Error("Expected error for invalid transport")
		}

		if !strings.Contains(err.Error(), "unsupported transport type") {
			t.Errorf("Expected unsupported transport error, got: %v", err)
		}
	})
}

// Helper function for creating bool pointers.
func boolPtr(b bool) *bool {
	return &b
}

// Mock implementations for testing

// MockServerFactory for testing.
type MockServerFactory struct {
	HTTPStartError   error
	StdioServeError  error
	HTTPStartCalled  bool
	StdioServeCalled bool
	HTTPStopCalled   bool
	StdioStopCalled  bool
}

func (f *MockServerFactory) CreateHTTPServer(mcpServer *server.MCPServer) HTTPServer {
	return &mockHTTPServer{factory: f}
}

func (f *MockServerFactory) CreateStdioServer(mcpServer *server.MCPServer) StdioServer {
	return &mockStdioServer{factory: f}
}

type mockHTTPServer struct {
	factory *MockServerFactory
}

func (s *mockHTTPServer) Start(addr string) error {
	s.factory.HTTPStartCalled = true
	return s.factory.HTTPStartError
}

func (s *mockHTTPServer) Stop() error {
	s.factory.HTTPStopCalled = true
	return nil
}

type mockStdioServer struct {
	factory *MockServerFactory
}

func (s *mockStdioServer) Serve() error {
	s.factory.StdioServeCalled = true
	return s.factory.StdioServeError
}

func (s *mockStdioServer) Stop() error {
	s.factory.StdioStopCalled = true
	return nil
}

// Mock implementation of MakeMCPTool for testing.
type mockMakeMCPTool struct{}

func (m *mockMakeMCPTool) GetName() string {
	return "mock_tool"
}

func (m *mockMakeMCPTool) GetHandler() core.MakeMcpToolHandler {
	return func(ctx context.Context, request core.ToolExecutionContext) (core.ToolExecutionResult, error) {
		return core.NewBasicExecutionResult("mock response", nil), nil
	}
}

func (m *mockMakeMCPTool) ToMcpTool() core.McpTool {
	return core.McpTool{
		Name:        "mock_tool",
		Description: "A mock tool for testing",
		InputSchema: core.McpToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
			Required:   []string{},
		},
		Annotations: core.McpToolAnnotation{
			Title: "Mock Tool",
		},
	}
}

func (m *mockMakeMCPTool) ToJSON() string {
	return `{"name":"mock_tool","description":"A mock tool for testing"}`
}
