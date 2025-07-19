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
	"encoding/json"
	"os"
	"strings"
	"testing"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
	"github.com/T4cceptor/MakeMCP/pkg/sources/openapi"
)

func TestSaveToFile(t *testing.T) {
	// Initialize registries for tests
	InitializeRegistries()

	tests := []struct {
		name        string
		app         *core.MakeMCPApp
		expectedExt string
		wantErr     bool
	}{
		{
			name: "save with custom filename",
			app: &core.MakeMCPApp{
				Name:       "test-app",
				Version:    "1.0.0",
				SourceType: "openapi",
				SourceParams: &openapi.OpenAPIParams{
					SharedParams: &core.SharedParams{
						File:       "test-config",
						Transport:  core.TransportTypeStdio,
						SourceType: "openapi",
					},
					Specs:   "http://example.com/openapi.json",
					BaseURL: "http://example.com",
					Timeout: 30,
				},
				Tools: []core.MakeMCPTool{},
			},
			expectedExt: "test-config.json",
			wantErr:     false,
		},
		{
			name: "save with default filename",
			app: &core.MakeMCPApp{
				Name:       "my-api",
				Version:    "1.0.0",
				SourceType: "openapi",
				SourceParams: &openapi.OpenAPIParams{
					SharedParams: &core.SharedParams{
						Transport:  core.TransportTypeHTTP,
						SourceType: "openapi",
					},
					Specs:   "http://example.com/openapi.json",
					BaseURL: "http://example.com",
					Timeout: 30,
				},
				Tools: []core.MakeMCPTool{},
			},
			expectedExt: "my-api_makemcp.json",
			wantErr:     false,
		},
		{
			name: "save with empty app name",
			app: &core.MakeMCPApp{
				Name:       "",
				Version:    "1.0.0",
				SourceType: "openapi",
				SourceParams: &openapi.OpenAPIParams{
					SharedParams: &core.SharedParams{
						Transport:  core.TransportTypeStdio,
						SourceType: "openapi",
					},
					Specs:   "http://example.com/openapi.json",
					BaseURL: "http://example.com",
					Timeout: 30,
				},
				Tools: []core.MakeMCPTool{},
			},
			expectedExt: "_makemcp.json",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing file
			if _, err := os.Stat(tt.expectedExt); err == nil {
				os.Remove(tt.expectedExt)
			}

			err := SaveToFile(tt.app)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveToFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file was created
				if _, err := os.Stat(tt.expectedExt); os.IsNotExist(err) {
					t.Errorf("Expected file %s was not created", tt.expectedExt)
					return
				}

				// Verify file contents using proper round-trip through LoadFromFile
				savedApp, err := LoadFromFile(tt.expectedExt)
				if err != nil {
					t.Errorf("Failed to load saved file via LoadFromFile: %v", err)
					return
				}

				// Compare key fields
				if savedApp.Name != tt.app.Name {
					t.Errorf("Name mismatch: got %s, want %s", savedApp.Name, tt.app.Name)
				}
				if savedApp.Version != tt.app.Version {
					t.Errorf("Version mismatch: got %s, want %s", savedApp.Version, tt.app.Version)
				}
				if savedApp.SourceType != tt.app.SourceType {
					t.Errorf("SourceType mismatch: got %s, want %s", savedApp.SourceType, tt.app.SourceType)
				}

				// Clean up
				os.Remove(tt.expectedExt)
			}
		})
	}
}

func TestSaveToFile_InvalidPath(t *testing.T) {
	app := &core.MakeMCPApp{
		Name:       "test",
		SourceType: "openapi",
		SourceParams: &openapi.OpenAPIParams{
			SharedParams: &core.SharedParams{
				File:       "/invalid/path/that/does/not/exist/config",
				SourceType: "openapi",
			},
			Specs:   "http://example.com/openapi.json",
			BaseURL: "http://example.com",
			Timeout: 30,
		},
	}

	err := SaveToFile(app)
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create file") {
		t.Errorf("Expected 'failed to create file' error, got: %v", err)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Initialize registries for tests
	InitializeRegistries()

	tests := []struct {
		name     string
		setup    func(t *testing.T) string // Returns filename
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, app *core.MakeMCPApp)
	}{
		{
			name: "load valid openapi config",
			setup: func(t *testing.T) string {
				// Create a JSON config manually to avoid interface marshaling issues
				configJSON := `{
  "name": "test-api",
  "version": "1.0.0",
  "sourceType": "openapi",
  "tools": [
    {
      "name": "get-users",
      "description": "Get all users",
      "inputSchema": {
        "type": "object"
      }
    }
  ],
  "config": {
    "transport": "http",
    "configOnly": false,
    "port": "8080",
    "devMode": false,
    "sourceType": "openapi",
    "file": "test",
    "specs": "http://localhost:8080/openapi.json",
    "baseURL": "http://localhost:8080",
    "timeout": 30,
    "strictValidate": false
  }
}`
				filename := "test_load_valid.json"
				if err := os.WriteFile(filename, []byte(configJSON), 0o644); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}

				return filename
			},
			wantErr: false,
			validate: func(t *testing.T, app *core.MakeMCPApp) {
				if app.Name != "test-api" {
					t.Errorf("Name mismatch: got %s, want test-api", app.Name)
				}
				if app.SourceType != "openapi" {
					t.Errorf("SourceType mismatch: got %s, want openapi", app.SourceType)
				}
				if len(app.Tools) != 1 {
					t.Errorf("Tools count mismatch: got %d, want 1", len(app.Tools))
				}
				if app.Tools[0].GetName() != "get-users" {
					t.Errorf("Tool name mismatch: got %s, want get-users", app.Tools[0].GetName())
				}
			},
		},
		{
			name: "load real makemcp.json structure",
			setup: func(t *testing.T) string {
				// Use the exact structure from the real makemcp.json file
				configJSON := `{
  "name": "FastAPI",
  "version": "0.1.0",
  "sourceType": "openapi",
  "tools": [
    {
      "name": "read_root__get",
      "description": "Root endpoint.\nReturns a welcome message from the FastAPI server.",
      "inputSchema": {
        "type": "object"
      },
      "annotations": {
        "title": "read_root__get",
        "readOnlyHint": true,
        "idempotentHint": true
      },
      "oapiHandlerInput": {
        "method": "GET",
        "path": "/",
        "headers": {},
        "cookies": {},
        "bodyAppend": {}
      }
    }
  ],
  "config": {
    "transport": "stdio",
    "configOnly": false,
    "port": "8080",
    "devMode": false,
    "sourceType": "openapi",
    "file": "makemcp",
    "specs": "http://localhost:8081/openapi.json",
    "baseURL": "http://localhost:8081",
    "timeout": 30,
    "strictValidate": false
  }
}`
				filename := "test_real_structure.json"
				if err := os.WriteFile(filename, []byte(configJSON), 0o644); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}

				return filename
			},
			wantErr: false,
			validate: func(t *testing.T, app *core.MakeMCPApp) {
				if app.Name != "FastAPI" {
					t.Errorf("Name mismatch: got %s, want FastAPI", app.Name)
				}
				if app.SourceType != "openapi" {
					t.Errorf("SourceType mismatch: got %s, want openapi", app.SourceType)
				}
				if len(app.Tools) != 1 {
					t.Errorf("Tools count mismatch: got %d, want 1", len(app.Tools))
				}
				if app.Tools[0].GetName() != "read_root__get" {
					t.Errorf("Tool name mismatch: got %s, want read_root__get", app.Tools[0].GetName())
				}
			},
		},
		{
			name: "file does not exist",
			setup: func(t *testing.T) string {
				return "nonexistent_file.json"
			},
			wantErr: true,
			errMsg:  "failed to open file",
		},
		{
			name: "invalid JSON",
			setup: func(t *testing.T) string {
				filename := "test_invalid_json.json"
				invalidJSON := `{"name": "test", "invalid": json}`
				if err := os.WriteFile(filename, []byte(invalidJSON), 0o644); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
				return filename
			},
			wantErr: true,
			errMsg:  "failed to decode metadata",
		},
		{
			name: "unknown source type",
			setup: func(t *testing.T) string {
				filename := "test_unknown_source.json"
				app := map[string]any{
					"name":       "test",
					"sourceType": "unknown-source-type",
				}
				data, _ := json.Marshal(app)
				if err := os.WriteFile(filename, data, 0o644); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
				return filename
			},
			wantErr: true,
			errMsg:  "unknown source type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.setup(t)
			defer func() {
				if _, err := os.Stat(filename); err == nil {
					os.Remove(filename)
				}
			}()

			app, err := LoadFromFile(filename)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if app == nil {
				t.Error("Expected non-nil app")
				return
			}

			if tt.validate != nil {
				tt.validate(t, app)
			}
		})
	}
}

func TestLoadFromFile_ReadError(t *testing.T) {
	// Create a directory instead of a file to trigger read error
	dirName := "test_directory"
	if err := os.Mkdir(dirName, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.Remove(dirName)

	_, err := LoadFromFile(dirName)
	if err == nil {
		t.Error("Expected error when trying to read directory as file")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	// Initialize registries for tests
	InitializeRegistries()

	originalApp := &core.MakeMCPApp{
		Name:       "roundtrip-test",
		Version:    "2.0.0",
		SourceType: "openapi",
		SourceParams: &openapi.OpenAPIParams{
			SharedParams: &core.SharedParams{
				File:       "roundtrip",
				Transport:  core.TransportTypeStdio,
				SourceType: "openapi",
			},
			Specs:   "http://localhost:8080/openapi.json",
			BaseURL: "http://localhost:8080",
			Timeout: 30,
		},
		Tools: []core.MakeMCPTool{},
	}

	filename := "roundtrip.json"
	defer func() {
		if _, err := os.Stat(filename); err == nil {
			os.Remove(filename)
		}
	}()

	// Save the app
	if err := SaveToFile(originalApp); err != nil {
		t.Fatalf("Failed to save app: %v", err)
	}

	// Load the app
	loadedApp, err := LoadFromFile(filename)
	if err != nil {
		t.Fatalf("Failed to load app: %v", err)
	}

	// Compare key fields
	if loadedApp.Name != originalApp.Name {
		t.Errorf("Name mismatch: got %s, want %s", loadedApp.Name, originalApp.Name)
	}
	if loadedApp.Version != originalApp.Version {
		t.Errorf("Version mismatch: got %s, want %s", loadedApp.Version, originalApp.Version)
	}
	if loadedApp.SourceType != originalApp.SourceType {
		t.Errorf("SourceType mismatch: got %s, want %s", loadedApp.SourceType, originalApp.SourceType)
	}
	if len(loadedApp.Tools) != len(originalApp.Tools) {
		t.Errorf("Tools count mismatch: got %d, want %d", len(loadedApp.Tools), len(originalApp.Tools))
	}
}
