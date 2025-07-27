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

package openapi

import (
	"testing"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
)

func TestOpenAPISource_LoadSpec(t *testing.T) {
	adapter := NewLibopenAPIAdapter()

	// Test only the valid case since LoadOpenAPISpec has proper error handling
	t.Run("Valid OpenAPI spec file", func(t *testing.T) {
		doc, err := adapter.LoadOpenAPISpec(
			"../../../testbed/openapi/sample_specifications/fastapi.json",
			false,
		)
		if err != nil {
			t.Fatalf("Expected no error but got: %v", err)
		}

		if doc == nil {
			t.Error("Expected valid OpenAPI document but got nil")
			return
		}

		// Validate basic structure using adapter
		title, version := adapter.GetDocumentInfo(doc)
		if title == "" {
			t.Error("Expected non-empty title")
		}
		if doc.Model.Paths == nil {
			t.Error("Expected Paths section but got nil")
		}

		// Check that it loaded our test data correctly
		if title != "FastAPI" {
			t.Errorf("Expected title 'FastAPI', got %s", title)
		}
		if version != "0.1.0" {
			t.Errorf("Expected version '0.1.0', got %s", version)
		}
	})
}

func TestOpenAPISource_Parse(t *testing.T) {
	source := &OpenAPISource{}

	// Create input parameters using new structure
	sharedParams := core.NewBaseParams("openapi", core.TransportTypeStdio)
	input := &core.CLIParamsInput{
		SharedParams: sharedParams,
		CliFlags: map[string]any{
			"specs":    "../../../testbed/openapi/sample_specifications/fastapi.json",
			"base-url": "http://localhost:8080",
		},
		CliArgs: []string{},
	}

	// Parse input into typed parameters
	appParams, err := source.ParseParams(input)
	if err != nil {
		t.Fatalf("Expected no error from ParseParams but got: %v", err)
	}

	app, err := source.Parse(appParams)
	if err != nil {
		t.Fatalf("Expected no error from Parse but got: %v", err)
	}

	// Test basic app structure
	if app.Name == "" {
		t.Error("Expected non-empty app name")
	}
	if app.Version == "" {
		t.Error("Expected non-empty app version")
	}
	if app.AppParams.GetSourceType() != "openapi" {
		t.Errorf("Expected source type 'openapi', got %s", app.AppParams.GetSourceType())
	}

	// Test tools generation - FastAPI spec has these operations
	expectedTools := []string{
		"read_root__get",
		"list_users_users_get",
		"create_user_users_post",
		"get_user_by_id_users__user_id__get",
		"update_user_users__user_id__patch",
		"delete_user_users__user_id__delete",
		"get_user_by_email_users_by_email__get",
	}

	if len(app.Tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(app.Tools))
	}

	// Check that all expected tools are present
	toolNames := make(map[string]bool)
	for _, tool := range app.Tools {
		toolNames[tool.GetName()] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolNames[expectedTool] {
			t.Errorf("Expected tool %s not found", expectedTool)
		}
	}
}

func TestOpenAPISource_Integration(t *testing.T) {
	// This test validates the complete integration flow
	// without testing private methods
	source := NewOpenAPISource()

	// Test the full cycle
	sharedParams := core.NewBaseParams("openapi", core.TransportTypeStdio)
	input := &core.CLIParamsInput{
		SharedParams: sharedParams,
		CliFlags: map[string]any{
			"specs":    "../../../testbed/openapi/sample_specifications/fastapi.json",
			"base-url": "http://localhost:8080",
		},
		CliArgs: []string{},
	}

	// Test parameter parsing
	appParams, err := source.ParseParams(input)
	if err != nil {
		t.Fatalf("Expected no error from ParseParams but got: %v", err)
	}

	// Test document parsing and tool generation
	app, err := source.Parse(appParams)
	if err != nil {
		t.Fatalf("Expected no error from Parse but got: %v", err)
	}

	// Validate the results
	if app == nil {
		t.Fatal("Expected valid app, got nil")
	}

	if len(app.Tools) == 0 {
		t.Error("Expected tools to be generated, got none")
	}

	// Validate tool structure
	for _, tool := range app.Tools {
		if tool.GetName() == "" {
			t.Error("Expected tool to have a name")
		}
		if tool.ToMcpTool().InputSchema.Type != "object" {
			t.Errorf("Expected tool input schema type 'object', got %s", tool.ToMcpTool().InputSchema.Type)
		}
	}
}

func TestOpenAPISource_Name(t *testing.T) {
	source := &OpenAPISource{}

	if source.Name() != "openapi" {
		t.Errorf("Expected name 'openapi', got %s", source.Name())
	}
}
