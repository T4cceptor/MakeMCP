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
	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPISource_LoadSpec(t *testing.T) {
	source := &OpenAPISource{}

	// Test only the valid case since loadOpenAPISpec calls log.Fatalf on error
	// which would exit the test process
	t.Run("Valid OpenAPI spec file", func(t *testing.T) {
		doc, err := source.loadOpenAPISpec(
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

		// Validate basic structure
		if doc.Info == nil {
			t.Error("Expected Info section but got nil")
		}
		if doc.Info.Title == "" {
			t.Error("Expected non-empty title")
		}
		if doc.Paths == nil {
			t.Error("Expected Paths section but got nil")
		}

		// Check that it loaded our test data correctly
		if doc.Info.Title != "FastAPI" {
			t.Errorf("Expected title 'FastAPI', got %s", doc.Info.Title)
		}
		if doc.Info.Version != "0.1.0" {
			t.Errorf("Expected version '0.1.0', got %s", doc.Info.Version)
		}
	})
}

func TestOpenAPISource_Parse(t *testing.T) {
	source := &OpenAPISource{}

	// Create input parameters using new structure
	sharedParams := core.NewSharedParams("openapi", core.TransportTypeStdio)
	input := &core.CLIParamsInput{
		SharedParams: sharedParams,
		CliFlags: map[string]any{
			"specs":    "../../../testbed/openapi/sample_specifications/fastapi.json",
			"base-url": "http://localhost:8080",
		},
		CliArgs: []string{},
	}

	// Parse input into typed parameters
	sourceParams, err := source.ParseParams(input)
	if err != nil {
		t.Fatalf("Expected no error from ParseParams but got: %v", err)
	}

	app, err := source.Parse(sourceParams)
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
	if app.SourceParams.GetSourceType() != "openapi" {
		t.Errorf("Expected source type 'openapi', got %s", app.SourceParams.GetSourceType())
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

func TestOpenAPISource_GetToolName(t *testing.T) {
	source := &OpenAPISource{}

	tests := []struct {
		name        string
		method      string
		path        string
		operationID string
		expected    string
	}{
		{
			name:        "With operation ID",
			method:      "GET",
			path:        "/users",
			operationID: "listUsers",
			expected:    "listusers",
		},
		{
			name:        "Without operation ID",
			method:      "POST",
			path:        "/users/{id}",
			operationID: "",
			expected:    "post__users_id",
		},
		{
			name:        "DELETE with operation ID",
			method:      "DELETE",
			path:        "/users/{id}",
			operationID: "deleteUser",
			expected:    "deleteuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operation := &openapi3.Operation{
				OperationID: tt.operationID,
			}

			result := source.getToolName(tt.method, tt.path, operation)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOpenAPISource_GetToolInputSchema(t *testing.T) {
	source := &OpenAPISource{}

	// Create a sample OpenAPI operation with various parameter types
	operation := &openapi3.Operation{
		Parameters: []*openapi3.ParameterRef{
			{
				Value: &openapi3.Parameter{
					Name:        "userId",
					In:          "path",
					Required:    true,
					Description: "User ID",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
						},
					},
				},
			},
			{
				Value: &openapi3.Parameter{
					Name:        "limit",
					In:          "query",
					Required:    false,
					Description: "Limit results",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"integer"},
						},
					},
				},
			},
			{
				Value: &openapi3.Parameter{
					Name:        "X-API-Key",
					In:          "header",
					Required:    true,
					Description: "API Key",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
						},
					},
				},
			},
		},
		RequestBody: &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Required: true,
				Content: map[string]*openapi3.MediaType{
					"application/json": {
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"object"},
								Properties: map[string]*openapi3.SchemaRef{
									"name": {
										Value: &openapi3.Schema{
											Type:        &openapi3.Types{"string"},
											Description: "User name",
										},
									},
									"email": {
										Value: &openapi3.Schema{
											Type:        &openapi3.Types{"string"},
											Description: "User email",
										},
									},
								},
								Required: []string{"name", "email"},
							},
						},
					},
				},
			},
		},
	}

	schema := source.getToolInputSchema(operation)

	// Check schema structure
	if schema.Type != "object" {
		t.Errorf("Expected schema type 'object', got %s", schema.Type)
	}

	// Check expected properties with prefixes
	expectedProperties := map[string]string{
		"path__userId":      "string",
		"query__limit":      "integer", // Should match the actual OpenAPI type
		"header__X-API-Key": "string",
		"body__name":        "string",
		"body__email":       "string",
	}

	for propName, expectedType := range expectedProperties {
		prop, exists := schema.Properties[propName]
		if !exists {
			t.Errorf("Expected property %s not found", propName)
			continue
		}

		propMap, ok := prop.(map[string]interface{})
		if !ok {
			t.Errorf("Property %s is not a map", propName)
			continue
		}

		if propMap["type"] != expectedType {
			t.Errorf("Expected property %s type %s, got %v", propName, expectedType, propMap["type"])
		}
	}

	// Check required fields
	expectedRequired := []string{"path__userId", "header__X-API-Key", "body__name", "body__email"}
	if len(schema.Required) != len(expectedRequired) {
		t.Errorf("Expected %d required fields, got %d", len(expectedRequired), len(schema.Required))
	}

	for _, req := range expectedRequired {
		found := false
		for _, actual := range schema.Required {
			if actual == req {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected required field %s not found", req)
		}
	}
}

func TestOpenAPISource_GetToolAnnotations(t *testing.T) {
	source := &OpenAPISource{}

	operation := &openapi3.Operation{
		Summary:     "Test operation",
		Description: "This is a test operation",
	}

	tests := []struct {
		name                string
		method              string
		path                string
		expectedReadOnly    *bool
		expectedDestructive *bool
		expectedIdempotent  *bool
	}{
		{
			name:                "GET method",
			method:              "GET",
			path:                "/users",
			expectedReadOnly:    boolPtr(true),
			expectedDestructive: nil,
			expectedIdempotent:  boolPtr(true),
		},
		{
			name:                "POST method",
			method:              "POST",
			path:                "/users",
			expectedReadOnly:    nil,
			expectedDestructive: nil,
			expectedIdempotent:  boolPtr(false),
		},
		{
			name:                "PUT method",
			method:              "PUT",
			path:                "/users/{id}",
			expectedReadOnly:    nil,
			expectedDestructive: nil,
			expectedIdempotent:  boolPtr(true),
		},
		{
			name:                "DELETE method",
			method:              "DELETE",
			path:                "/users/{id}",
			expectedReadOnly:    nil,
			expectedDestructive: boolPtr(true),
			expectedIdempotent:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := source.getToolAnnotations(tt.method, tt.path, operation)

			if tt.expectedReadOnly != nil {
				if annotations.ReadOnlyHint == nil || *annotations.ReadOnlyHint != *tt.expectedReadOnly {
					t.Errorf("Expected ReadOnlyHint %v, got %v", *tt.expectedReadOnly, annotations.ReadOnlyHint)
				}
			} else if annotations.ReadOnlyHint != nil {
				t.Errorf("Expected ReadOnlyHint nil, got %v", *annotations.ReadOnlyHint)
			}

			if tt.expectedDestructive != nil {
				if annotations.DestructiveHint == nil || *annotations.DestructiveHint != *tt.expectedDestructive {
					t.Errorf("Expected DestructiveHint %v, got %v", *tt.expectedDestructive, annotations.DestructiveHint)
				}
			} else if annotations.DestructiveHint != nil {
				t.Errorf("Expected DestructiveHint nil, got %v", *annotations.DestructiveHint)
			}

			if tt.expectedIdempotent != nil {
				if annotations.IdempotentHint == nil || *annotations.IdempotentHint != *tt.expectedIdempotent {
					t.Errorf("Expected IdempotentHint %v, got %v", *tt.expectedIdempotent, annotations.IdempotentHint)
				}
			} else if annotations.IdempotentHint != nil {
				t.Errorf("Expected IdempotentHint nil, got %v", *annotations.IdempotentHint)
			}
		})
	}
}

func TestOpenAPISource_DetectSourceType(t *testing.T) {
	source := &OpenAPISource{}

	tests := []struct {
		name     string
		location string
		expected string
	}{
		{
			name:     "HTTP URL",
			location: "http://example.com/openapi.json",
			expected: "url",
		},
		{
			name:     "HTTPS URL",
			location: "https://example.com/openapi.json",
			expected: "url",
		},
		{
			name:     "File path",
			location: "/path/to/openapi.json",
			expected: "file",
		},
		{
			name:     "Relative file path",
			location: "openapi.json",
			expected: "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := source.detectSourceType(tt.location)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOpenAPISource_Name(t *testing.T) {
	source := &OpenAPISource{}

	if source.Name() != "openapi" {
		t.Errorf("Expected name 'openapi', got %s", source.Name())
	}
}
