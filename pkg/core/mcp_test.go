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
	"encoding/json"
	"testing"
)

func TestTransportType_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		transport TransportType
		want      bool
	}{
		{
			name:      "HTTP transport is valid",
			transport: TransportTypeHTTP,
			want:      true,
		},
		{
			name:      "Stdio transport is valid",
			transport: TransportTypeStdio,
			want:      true,
		},
		{
			name:      "Empty transport is invalid",
			transport: TransportType(""),
			want:      false,
		},
		{
			name:      "Unknown transport is invalid",
			transport: TransportType("websocket"),
			want:      false,
		},
		{
			name:      "Random string is invalid",
			transport: TransportType("invalid"),
			want:      false,
		},
		{
			name:      "Case sensitive - HTTP uppercase invalid",
			transport: TransportType("HTTP"),
			want:      false,
		},
		{
			name:      "Case sensitive - STDIO uppercase invalid",
			transport: TransportType("STDIO"),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.transport.IsValid(); got != tt.want {
				t.Errorf("TransportType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransportType_Constants(t *testing.T) {
	// Test that constants have expected values
	if TransportTypeHTTP != "http" {
		t.Errorf("TransportTypeHTTP = %v, want 'http'", TransportTypeHTTP)
	}
	if TransportTypeStdio != "stdio" {
		t.Errorf("TransportTypeStdio = %v, want 'stdio'", TransportTypeStdio)
	}
}

func TestMcpToolInputSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema McpToolInputSchema
	}{
		{
			name: "basic schema",
			schema: McpToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "User name",
					},
					"age": map[string]any{
						"type": "integer",
					},
				},
				Required: []string{"name"},
			},
		},
		{
			name: "empty schema",
			schema: McpToolInputSchema{
				Type: "object",
			},
		},
		{
			name: "complex schema",
			schema: McpToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"nested": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"field": map[string]any{
								"type": "string",
							},
						},
					},
				},
				Required: []string{"nested"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling/unmarshaling
			data, err := json.Marshal(tt.schema)
			if err != nil {
				t.Errorf("Failed to marshal schema: %v", err)
				return
			}

			var unmarshaled McpToolInputSchema
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Errorf("Failed to unmarshal schema: %v", err)
				return
			}

			if unmarshaled.Type != tt.schema.Type {
				t.Errorf("Type mismatch: got %v, want %v", unmarshaled.Type, tt.schema.Type)
			}

			if len(unmarshaled.Required) != len(tt.schema.Required) {
				t.Errorf("Required length mismatch: got %d, want %d", len(unmarshaled.Required), len(tt.schema.Required))
			}

			for i, req := range tt.schema.Required {
				if i < len(unmarshaled.Required) && unmarshaled.Required[i] != req {
					t.Errorf("Required[%d] mismatch: got %v, want %v", i, unmarshaled.Required[i], req)
				}
			}
		})
	}
}

func TestMcpToolAnnotation(t *testing.T) {
	tests := []struct {
		name       string
		annotation McpToolAnnotation
	}{
		{
			name: "all fields set",
			annotation: McpToolAnnotation{
				Title:           "Test Tool",
				ReadOnlyHint:    boolPtr(true),
				DestructiveHint: boolPtr(false),
				IdempotentHint:  boolPtr(true),
				OpenWorldHint:   boolPtr(false),
			},
		},
		{
			name: "minimal annotation",
			annotation: McpToolAnnotation{
				Title: "Simple Tool",
			},
		},
		{
			name: "only hints",
			annotation: McpToolAnnotation{
				ReadOnlyHint:   boolPtr(false),
				IdempotentHint: boolPtr(true),
			},
		},
		{
			name:       "empty annotation",
			annotation: McpToolAnnotation{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling/unmarshaling
			data, err := json.Marshal(tt.annotation)
			if err != nil {
				t.Errorf("Failed to marshal annotation: %v", err)
				return
			}

			var unmarshaled McpToolAnnotation
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Errorf("Failed to unmarshal annotation: %v", err)
				return
			}

			if unmarshaled.Title != tt.annotation.Title {
				t.Errorf("Title mismatch: got %v, want %v", unmarshaled.Title, tt.annotation.Title)
			}

			// Test pointer comparisons
			if !boolPtrEqual(unmarshaled.ReadOnlyHint, tt.annotation.ReadOnlyHint) {
				t.Errorf("ReadOnlyHint mismatch: got %v, want %v", unmarshaled.ReadOnlyHint, tt.annotation.ReadOnlyHint)
			}
			if !boolPtrEqual(unmarshaled.DestructiveHint, tt.annotation.DestructiveHint) {
				t.Errorf("DestructiveHint mismatch: got %v, want %v", unmarshaled.DestructiveHint, tt.annotation.DestructiveHint)
			}
			if !boolPtrEqual(unmarshaled.IdempotentHint, tt.annotation.IdempotentHint) {
				t.Errorf("IdempotentHint mismatch: got %v, want %v", unmarshaled.IdempotentHint, tt.annotation.IdempotentHint)
			}
			if !boolPtrEqual(unmarshaled.OpenWorldHint, tt.annotation.OpenWorldHint) {
				t.Errorf("OpenWorldHint mismatch: got %v, want %v", unmarshaled.OpenWorldHint, tt.annotation.OpenWorldHint)
			}
		})
	}
}

func TestMcpTool(t *testing.T) {
	tests := []struct {
		name string
		tool McpTool
	}{
		{
			name: "complete tool",
			tool: McpTool{
				Name:        "test-tool",
				Description: "A test tool",
				InputSchema: McpToolInputSchema{
					Type: "object",
					Properties: map[string]any{
						"param1": map[string]any{"type": "string"},
					},
					Required: []string{"param1"},
				},
				Annotations: McpToolAnnotation{
					Title:          "Test Tool",
					ReadOnlyHint:   boolPtr(true),
					IdempotentHint: boolPtr(true),
				},
			},
		},
		{
			name: "minimal tool",
			tool: McpTool{
				Name: "simple-tool",
				InputSchema: McpToolInputSchema{
					Type: "object",
				},
				Annotations: McpToolAnnotation{},
			},
		},
		{
			name: "tool with raw input schema",
			tool: McpTool{
				Name:           "raw-tool",
				RawInputSchema: json.RawMessage(`{"type": "string"}`),
				InputSchema: McpToolInputSchema{
					Type: "object",
				},
				Annotations: McpToolAnnotation{
					Title: "Raw Tool",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.tool)
			if err != nil {
				t.Errorf("Failed to marshal tool: %v", err)
				return
			}

			// Verify RawInputSchema is not included in JSON (has json:"-" tag)
			var jsonData map[string]any
			if err := json.Unmarshal(data, &jsonData); err != nil {
				t.Errorf("Failed to unmarshal to map: %v", err)
				return
			}

			if _, exists := jsonData["RawInputSchema"]; exists {
				t.Error("RawInputSchema should not be included in JSON output")
			}

			// Test JSON unmarshaling
			var unmarshaled McpTool
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Errorf("Failed to unmarshal tool: %v", err)
				return
			}

			if unmarshaled.Name != tt.tool.Name {
				t.Errorf("Name mismatch: got %v, want %v", unmarshaled.Name, tt.tool.Name)
			}
			if unmarshaled.Description != tt.tool.Description {
				t.Errorf("Description mismatch: got %v, want %v", unmarshaled.Description, tt.tool.Description)
			}
		})
	}
}

// Helper functions for tests
func boolPtr(b bool) *bool {
	return &b
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}