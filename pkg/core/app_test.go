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
	"testing"
)

// mockAppParams implements AppParams for testing.
type mockAppParams struct {
	sharedParams *BaseAppParams
	sourceType   string
}

func (m *mockAppParams) GetSharedParams() *BaseAppParams {
	return m.sharedParams
}

func (m *mockAppParams) Validate() error {
	return nil
}

func (m *mockAppParams) ToJSON() string {
	return `{"sourceType": "` + m.sourceType + `"}`
}

func (m *mockAppParams) GetSourceType() string {
	return m.sourceType
}

// mockTool implements MakeMCPTool for testing.
type mockTool struct {
	name        string
	description string
}

func (m *mockTool) GetName() string {
	return m.name
}

func (m *mockTool) GetHandler() MakeMcpToolHandler {
	return func(ctx context.Context, request ToolExecutionContext) (ToolExecutionResult, error) {
		return NewBasicExecutionResult("mock response", nil), nil
	}
}

func (m *mockTool) ToMcpTool() McpTool {
	return McpTool{
		Name:        m.name,
		Description: m.description,
		InputSchema: McpToolInputSchema{
			Type: "object",
		},
		Annotations: McpToolAnnotation{
			Title: m.name,
		},
	}
}

func (m *mockTool) ToJSON() string {
	return `{"name": "` + m.name + `", "description": "` + m.description + `"}`
}

func TestNewMakeMCPApp(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		version     string
		appParams   AppParams
		wantName    string
		wantVersion string
		wantSource  string
	}{
		{
			name:    "create app with basic params",
			appName: "TestApp",
			version: "1.0.0",
			appParams: &mockAppParams{
				sharedParams: NewBaseParams("test", TransportTypeStdio),
				sourceType:   "test",
			},
			wantName:    "TestApp",
			wantVersion: "1.0.0",
			wantSource:  "test",
		},
		{
			name:    "create app with empty name",
			appName: "",
			version: "2.0.0",
			appParams: &mockAppParams{
				sharedParams: NewBaseParams("openapi", TransportTypeHTTP),
				sourceType:   "openapi",
			},
			wantName:    "",
			wantVersion: "2.0.0",
			wantSource:  "openapi",
		},
		{
			name:    "create app with different transport",
			appName: "HTTPApp",
			version: "1.2.3",
			appParams: &mockAppParams{
				sharedParams: NewBaseParams("cli", TransportTypeHTTP),
				sourceType:   "cli",
			},
			wantName:    "HTTPApp",
			wantVersion: "1.2.3",
			wantSource:  "cli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewMakeMCPApp(tt.appName, tt.version, tt.appParams)

			if app.Name != tt.wantName {
				t.Errorf("NewMakeMCPApp().Name = %v, want %v", app.Name, tt.wantName)
			}
			if app.Version != tt.wantVersion {
				t.Errorf("NewMakeMCPApp().Version = %v, want %v", app.Version, tt.wantVersion)
			}
			if app.SourceType != tt.wantSource {
				t.Errorf("NewMakeMCPApp().SourceType = %v, want %v", app.SourceType, tt.wantSource)
			}
			if app.AppParams != tt.appParams {
				t.Errorf("NewMakeMCPApp().AppParams = %v, want %v", app.AppParams, tt.appParams)
			}
			if len(app.Tools) != 0 {
				t.Errorf("NewMakeMCPApp().Tools should be empty, got %d tools", len(app.Tools))
			}
		})
	}
}

func TestMakeMCPApp_ToolsManagement(t *testing.T) {
	appParams := &mockAppParams{
		sharedParams: NewBaseParams("test", TransportTypeStdio),
		sourceType:   "test",
	}

	app := NewMakeMCPApp("TestApp", "1.0.0", appParams)

	// Test adding tools
	tool1 := &mockTool{name: "tool1", description: "First tool"}
	tool2 := &mockTool{name: "tool2", description: "Second tool"}

	app.Tools = append(app.Tools, tool1, tool2)

	if len(app.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(app.Tools))
	}

	if app.Tools[0].GetName() != "tool1" {
		t.Errorf("Expected first tool name 'tool1', got %s", app.Tools[0].GetName())
	}

	if app.Tools[1].GetName() != "tool2" {
		t.Errorf("Expected second tool name 'tool2', got %s", app.Tools[1].GetName())
	}
}

func TestMakeMCPApp_SourceTypeFromParams(t *testing.T) {
	tests := []struct {
		name       string
		sourceType string
	}{
		{"openapi source", "openapi"},
		{"cli source", "cli"},
		{"custom source", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appParams := &mockAppParams{
				sharedParams: NewBaseParams(tt.sourceType, TransportTypeStdio),
				sourceType:   tt.sourceType,
			}

			app := NewMakeMCPApp("TestApp", "1.0.0", appParams)

			if app.SourceType != tt.sourceType {
				t.Errorf("Expected SourceType %s, got %s", tt.sourceType, app.SourceType)
			}
		})
	}
}
