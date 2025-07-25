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
	"testing"
)

// testSourceParams implements SourceParams for testing.
type testSourceParams struct {
	sharedParams *BaseAppParams
	CustomField  string `json:"customField"`
}

func (t *testSourceParams) GetSharedParams() *BaseAppParams {
	return t.sharedParams
}

func (t *testSourceParams) Validate() error {
	return nil
}

func (t *testSourceParams) ToJSON() string {
	data, _ := json.Marshal(t)
	return string(data)
}

func (t *testSourceParams) GetSourceType() string {
	return "test"
}

// testTool implements MakeMCPTool for testing.
type testTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (t *testTool) GetName() string {
	return t.Name
}

func (t *testTool) GetHandler() MakeMcpToolHandler {
	return func(ctx context.Context, request ToolExecutionContext) (ToolExecutionResult, error) {
		return NewBasicExecutionResult("test response", nil), nil
	}
}

func (t *testTool) ToMcpTool() McpTool {
	return McpTool{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: McpToolInputSchema{
			Type: "object",
		},
		Annotations: McpToolAnnotation{
			Title: t.Name,
		},
	}
}

func (t *testTool) ToJSON() string {
	data, _ := json.Marshal(t)
	return string(data)
}

func TestUnmarshalConfigWithTypedParams_Success(t *testing.T) {
	tests := []struct {
		name       string
		jsonData   string
		wantName   string
		wantSource string
		wantTools  int
	}{
		{
			name: "complete config with tools",
			jsonData: `{
				"name": "TestApp",
				"version": "1.0.0",
				"sourceType": "test",
				"tools": [
					{
						"name": "tool1",
						"description": "First tool"
					},
					{
						"name": "tool2",
						"description": "Second tool"
					}
				],
				"config": {
					"transport": "stdio",
					"configOnly": false,
					"port": "8080",
					"devMode": false,
					"sourceType": "test",
					"file": "test-config",
					"customField": "test-value"
				}
			}`,
			wantName:   "TestApp",
			wantSource: "test",
			wantTools:  2,
		},
		{
			name: "minimal config",
			jsonData: `{
				"name": "MinimalApp",
				"version": "0.1.0",
				"sourceType": "test",
				"tools": [],
				"config": {
					"transport": "http",
					"sourceType": "test",
					"customField": "minimal"
				}
			}`,
			wantName:   "MinimalApp",
			wantSource: "test",
			wantTools:  0,
		},
		{
			name: "config with single tool",
			jsonData: `{
				"name": "SingleToolApp",
				"version": "2.0.0",
				"sourceType": "test",
				"tools": [
					{
						"name": "single-tool",
						"description": "Only tool"
					}
				],
				"config": {
					"transport": "stdio",
					"sourceType": "test",
					"customField": "single"
				}
			}`,
			wantName:   "SingleToolApp",
			wantSource: "test",
			wantTools:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := UnmarshalConfigWithTypedParams[*testTool, *testSourceParams]([]byte(tt.jsonData))
			if err != nil {
				t.Errorf("UnmarshalConfigWithTypedParams() error = %v", err)
				return
			}

			if app == nil {
				t.Error("UnmarshalConfigWithTypedParams() returned nil app")
				return
			}

			if app.Name != tt.wantName {
				t.Errorf("App.Name = %v, want %v", app.Name, tt.wantName)
			}

			if app.SourceType != tt.wantSource {
				t.Errorf("App.SourceType = %v, want %v", app.SourceType, tt.wantSource)
			}

			if len(app.Tools) != tt.wantTools {
				t.Errorf("len(App.Tools) = %v, want %v", len(app.Tools), tt.wantTools)
			}

			// Verify tools are correctly converted to interface
			for i, tool := range app.Tools {
				if tool == nil {
					t.Errorf("App.Tools[%d] is nil", i)
					continue
				}
				if tool.GetName() == "" {
					t.Errorf("App.Tools[%d].GetName() is empty", i)
				}
			}

			// Verify source params are correctly set
			if app.SourceParams == nil {
				t.Error("App.SourceParams is nil")
			} else if app.SourceParams.GetSourceType() != tt.wantSource {
				t.Errorf("App.SourceParams.GetSourceType() = %v, want %v",
					app.SourceParams.GetSourceType(), tt.wantSource)
			}
		})
	}
}

func TestUnmarshalConfigWithTypedParams_InvalidJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "invalid JSON syntax",
			jsonData: `{"name": "test", "invalid": json}`,
			wantErr:  true,
		},
		{
			name:     "empty JSON",
			jsonData: ``,
			wantErr:  true,
		},
		{
			name:     "malformed array",
			jsonData: `{"tools": [{"name": "test",}]}`,
			wantErr:  true,
		},
		{
			name:     "wrong type for tools",
			jsonData: `{"tools": "not-an-array"}`,
			wantErr:  true,
		},
		{
			name:     "wrong type for config",
			jsonData: `{"config": "not-an-object"}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalConfigWithTypedParams[*testTool, *testSourceParams]([]byte(tt.jsonData))
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalConfigWithTypedParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnmarshalConfigWithTypedParams_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		checkFn  func(*testing.T, *MakeMCPApp)
	}{
		{
			name: "missing name",
			jsonData: `{
				"version": "1.0.0",
				"sourceType": "test",
				"tools": [],
				"config": {
					"transport": "stdio",
					"sourceType": "test",
					"customField": "test"
				}
			}`,
			checkFn: func(t *testing.T, app *MakeMCPApp) {
				if app.Name != "" {
					t.Errorf("Expected empty name, got %v", app.Name)
				}
			},
		},
		{
			name: "missing version",
			jsonData: `{
				"name": "TestApp",
				"sourceType": "test",
				"tools": [],
				"config": {
					"transport": "stdio",
					"sourceType": "test",
					"customField": "test"
				}
			}`,
			checkFn: func(t *testing.T, app *MakeMCPApp) {
				if app.Version != "" {
					t.Errorf("Expected empty version, got %v", app.Version)
				}
			},
		},
		{
			name: "missing tools",
			jsonData: `{
				"name": "TestApp",
				"version": "1.0.0",
				"sourceType": "test",
				"config": {
					"transport": "stdio",
					"sourceType": "test",
					"customField": "test"
				}
			}`,
			checkFn: func(t *testing.T, app *MakeMCPApp) {
				if len(app.Tools) != 0 {
					t.Errorf("Expected empty tools, got %d tools", len(app.Tools))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := UnmarshalConfigWithTypedParams[*testTool, *testSourceParams]([]byte(tt.jsonData))
			if err != nil {
				t.Errorf("UnmarshalConfigWithTypedParams() unexpected error = %v", err)
				return
			}

			if app == nil {
				t.Error("UnmarshalConfigWithTypedParams() returned nil app")
				return
			}

			tt.checkFn(t, app)
		})
	}
}

func TestUnmarshalConfigWithTypedParams_TypeConversion(t *testing.T) {
	jsonData := `{
		"name": "TypeTest",
		"version": "1.0.0",
		"sourceType": "test",
		"tools": [
			{
				"name": "typed-tool",
				"description": "A tool for type testing"
			}
		],
		"config": {
			"transport": "http",
			"configOnly": true,
			"port": "9090",
			"devMode": true,
			"sourceType": "test",
			"file": "type-test",
			"customField": "type-value"
		}
	}`

	app, err := UnmarshalConfigWithTypedParams[*testTool, *testSourceParams]([]byte(jsonData))
	if err != nil {
		t.Fatalf("UnmarshalConfigWithTypedParams() error = %v", err)
	}

	// Test tool type conversion
	if len(app.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(app.Tools))
	}

	tool := app.Tools[0]
	if tool.GetName() != "typed-tool" {
		t.Errorf("Tool name = %v, want 'typed-tool'", tool.GetName())
	}

	// Verify the tool can be converted back to McpTool
	mcpTool := tool.ToMcpTool()
	if mcpTool.Name != "typed-tool" {
		t.Errorf("McpTool name = %v, want 'typed-tool'", mcpTool.Name)
	}

	// Test source params type conversion
	sourceParams := app.SourceParams
	if sourceParams == nil {
		t.Fatal("SourceParams is nil")
	}

	if sourceParams.GetSourceType() != "test" {
		t.Errorf("SourceType = %v, want 'test'", sourceParams.GetSourceType())
	}

	// Type assert to check the concrete type
	typedParams, ok := sourceParams.(*testSourceParams)
	if !ok {
		t.Errorf("SourceParams is not *testSourceParams, got %T", sourceParams)
	} else if typedParams.CustomField != "type-value" {
		t.Errorf("CustomField = %v, want 'type-value'", typedParams.CustomField)
	}
}
