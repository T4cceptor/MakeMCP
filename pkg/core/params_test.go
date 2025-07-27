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

func TestNewSharedParams(t *testing.T) {
	tests := []struct {
		name          string
		sourceType    string
		transport     TransportType
		wantType      string
		wantTransport TransportType
	}{
		{
			name:          "valid stdio transport",
			sourceType:    "openapi",
			transport:     TransportTypeStdio,
			wantType:      "openapi",
			wantTransport: TransportTypeStdio,
		},
		{
			name:          "valid http transport",
			sourceType:    "cli",
			transport:     TransportTypeHTTP,
			wantType:      "cli",
			wantTransport: TransportTypeHTTP,
		},
		{
			name:          "invalid transport defaults to stdio",
			sourceType:    "test",
			transport:     TransportType("invalid"),
			wantType:      "test",
			wantTransport: TransportTypeStdio,
		},
		{
			name:          "empty transport defaults to stdio",
			sourceType:    "custom",
			transport:     TransportType(""),
			wantType:      "custom",
			wantTransport: TransportTypeStdio,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := NewBaseParams(tt.sourceType, tt.transport)

			if params.SourceType != tt.wantType {
				t.Errorf("NewSharedParams().SourceType = %v, want %v", params.SourceType, tt.wantType)
			}
			if params.Transport != tt.wantTransport {
				t.Errorf("NewSharedParams().Transport = %v, want %v", params.Transport, tt.wantTransport)
			}

			// Test default values
			if params.ConfigOnly != false {
				t.Errorf("NewSharedParams().ConfigOnly = %v, want false", params.ConfigOnly)
			}
			if params.Port != "8080" {
				t.Errorf("NewSharedParams().Port = %v, want '8080'", params.Port)
			}
			if params.DevMode != false {
				t.Errorf("NewSharedParams().DevMode = %v, want false", params.DevMode)
			}
			if params.File != "makemcp" {
				t.Errorf("NewSharedParams().File = %v, want 'makemcp'", params.File)
			}
		})
	}
}

func TestCLIParamsInput_ToJSON(t *testing.T) {
	tests := []struct {
		name  string
		input CLIParamsInput
	}{
		{
			name: "complete input",
			input: CLIParamsInput{
				SharedParams: &BaseAppParams{
					Transport:  TransportTypeHTTP,
					ConfigOnly: false,
					Port:       "8080",
					DevMode:    false,
					SourceType: "openapi",
					File:       "test",
				},
				CliFlags: map[string]any{
					"base-url": "http://localhost:8080",
					"timeout":  30,
					"strict":   true,
				},
				CliArgs: []string{"arg1", "arg2"},
			},
		},
		{
			name: "minimal input",
			input: CLIParamsInput{
				SharedParams: &BaseAppParams{
					Transport:  TransportTypeStdio,
					SourceType: "test",
				},
				CliFlags: map[string]any{},
				CliArgs:  []string{},
			},
		},
		{
			name: "nil shared params",
			input: CLIParamsInput{
				SharedParams: nil,
				CliFlags: map[string]any{
					"flag1": "value1",
				},
				CliArgs: []string{"arg"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr := tt.input.ToJSON()

			// Verify it's valid JSON
			var result map[string]any
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				t.Errorf("ToJSON() returned invalid JSON: %v", err)
				return
			}

			// Verify structure
			if _, exists := result["shared"]; !exists {
				t.Error("JSON should contain 'shared' field")
			}
			if _, exists := result["flags"]; !exists {
				t.Error("JSON should contain 'flags' field")
			}
			if _, exists := result["args"]; !exists {
				t.Error("JSON should contain 'args' field")
			}

			// Verify args array
			args, ok := result["args"].([]any)
			if !ok {
				t.Error("JSON args should be an array")
			} else if len(args) != len(tt.input.CliArgs) {
				t.Errorf("JSON args length = %d, want %d", len(args), len(tt.input.CliArgs))
			}

			// Verify flags object
			flags, ok := result["flags"].(map[string]any)
			if !ok {
				t.Error("JSON flags should be an object")
			} else if len(flags) != len(tt.input.CliFlags) {
				t.Errorf("JSON flags length = %d, want %d", len(flags), len(tt.input.CliFlags))
			}
		})
	}
}

func TestSharedParams_JSONRoundTrip(t *testing.T) {
	original := BaseAppParams{
		Transport:  TransportTypeHTTP,
		ConfigOnly: true,
		Port:       "9090",
		DevMode:    true,
		SourceType: "openapi",
		File:       "custom-config",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal SharedParams: %v", err)
	}

	// Unmarshal from JSON
	var unmarshaled BaseAppParams
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal SharedParams: %v", err)
	}

	// Compare all fields
	if unmarshaled.Transport != original.Transport {
		t.Errorf("Transport mismatch: got %v, want %v", unmarshaled.Transport, original.Transport)
	}
	if unmarshaled.ConfigOnly != original.ConfigOnly {
		t.Errorf("ConfigOnly mismatch: got %v, want %v", unmarshaled.ConfigOnly, original.ConfigOnly)
	}
	if unmarshaled.Port != original.Port {
		t.Errorf("Port mismatch: got %v, want %v", unmarshaled.Port, original.Port)
	}
	if unmarshaled.DevMode != original.DevMode {
		t.Errorf("DevMode mismatch: got %v, want %v", unmarshaled.DevMode, original.DevMode)
	}
	if unmarshaled.SourceType != original.SourceType {
		t.Errorf("SourceType mismatch: got %v, want %v", unmarshaled.SourceType, original.SourceType)
	}
	if unmarshaled.File != original.File {
		t.Errorf("File mismatch: got %v, want %v", unmarshaled.File, original.File)
	}
}

func TestCLIParamsInput_JSONRoundTrip(t *testing.T) {
	original := CLIParamsInput{
		SharedParams: &BaseAppParams{
			Transport:  TransportTypeStdio,
			ConfigOnly: false,
			Port:       "8080",
			DevMode:    true,
			SourceType: "test",
			File:       "roundtrip",
		},
		CliFlags: map[string]any{
			"string-flag": "value",
			"int-flag":    42,
			"bool-flag":   true,
		},
		CliArgs: []string{"arg1", "arg2", "arg3"},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal CLIParamsInput: %v", err)
	}

	// Unmarshal from JSON
	var unmarshaled CLIParamsInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal CLIParamsInput: %v", err)
	}

	// Compare shared params
	if unmarshaled.SharedParams == nil {
		t.Fatal("SharedParams should not be nil")
	}
	if unmarshaled.SharedParams.SourceType != original.SharedParams.SourceType {
		t.Errorf("SharedParams.SourceType mismatch: got %v, want %v",
			unmarshaled.SharedParams.SourceType, original.SharedParams.SourceType)
	}

	// Compare args
	if len(unmarshaled.CliArgs) != len(original.CliArgs) {
		t.Errorf("CliArgs length mismatch: got %d, want %d", len(unmarshaled.CliArgs), len(original.CliArgs))
	}

	// Compare flags
	if len(unmarshaled.CliFlags) != len(original.CliFlags) {
		t.Errorf("CliFlags length mismatch: got %d, want %d", len(unmarshaled.CliFlags), len(original.CliFlags))
	}
}
