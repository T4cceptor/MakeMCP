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

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Helper function to create a test OpenAPI document with a specific operation
func createTestOperation(requestBody string) (*v3.Operation, error) {
	// Create a minimal OpenAPI spec with the given request body
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    post:
      requestBody:
` + requestBody + `
      responses:
        '200':
          description: Success
`

	config := datamodel.NewDocumentConfiguration()
	document, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
	if err != nil {
		return nil, err
	}

	docModel, errors := document.BuildV3Model()
	if len(errors) > 0 {
		return nil, errors[0]
	}

	// Get the POST operation from /test path
	for pathPairs := docModel.Model.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
		pathItem := pathPairs.Value()
		if pathItem.Post != nil {
			return pathItem.Post, nil
		}
	}

	return nil, nil
}

// Helper function to create a test OpenAPIMcpTool with a specific request body
func createTestTool(requestBody string) (*OpenAPIMcpTool, error) {
	operation, err := createTestOperation(requestBody)
	if err != nil {
		return nil, err
	}
	if operation == nil {
		return nil, nil
	}

	// Determine content type from the operation's request body
	contentType := "application/json" // default
	if operation.RequestBody != nil && operation.RequestBody.Content != nil {
		// Use the first available content type
		contentType = operation.RequestBody.Content.First().Key()
	}

	tool := &OpenAPIMcpTool{
		Operation: operation,
		OpenAPIHandlerInput: &OpenAPIHandlerInput{
			Method:      "POST",
			Path:        "/test",
			ContentType: contentType,
		},
	}
	return tool, nil
}

func TestExtractRequestBodyProperties_JSON(t *testing.T) {
	adapter := NewLibopenAPIAdapter()

	tests := []struct {
		name          string
		requestBody   string
		expectedProps map[string]ToolInputProperty
		expectedReqs  []string
		description   string
	}{
		{
			name: "JSON with properties and required fields",
			requestBody: `
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                  description: User name
                age:
                  type: integer
                  description: User age
                email:
                  type: string
                  description: User email
              required: [name, email]`,
			expectedProps: map[string]ToolInputProperty{
				"name": {
					Type:        "string",
					Description: "User name",
					Location:    "body",
				},
				"age": {
					Type:        "integer",
					Description: "User age",
					Location:    "body",
				},
				"email": {
					Type:        "string",
					Description: "User email",
					Location:    "body",
				},
			},
			expectedReqs: []string{"name", "email"},
			description:  "Should extract individual JSON properties with proper types and required fields",
		},
		{
			name: "JSON with nested object",
			requestBody: `
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                user:
                  type: object
                  description: User object
                  properties:
                    name:
                      type: string
                    age:
                      type: integer
                metadata:
                  type: string
                  description: Additional metadata
              required: [user]`,
			expectedProps: map[string]ToolInputProperty{
				"user": {
					Type:        "object",
					Description: "User object",
					Location:    "body",
				},
				"metadata": {
					Type:        "string",
					Description: "Additional metadata",
					Location:    "body",
				},
			},
			expectedReqs: []string{"user"},
			description:  "Should handle nested objects as single properties",
		},
		{
			name: "JSON with array property",
			requestBody: `
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                items:
                  type: array
                  description: List of items
                  items:
                    type: string
                count:
                  type: integer
                  description: Number of items
              required: [items]`,
			expectedProps: map[string]ToolInputProperty{
				"items": {
					Type:        "array",
					Description: "List of items",
					Location:    "body",
				},
				"count": {
					Type:        "integer",
					Description: "Number of items",
					Location:    "body",
				},
			},
			expectedReqs: []string{"items"},
			description:  "Should handle array properties correctly",
		},
		{
			name: "Generic content type (*/*)",
			requestBody: `
        required: true
        content:
          "*/*":
            schema:
              type: object
              properties:
                data:
                  type: string
                  description: Generic data
              required: [data]`,
			expectedProps: map[string]ToolInputProperty{
				"data": {
					Type:        "string",
					Description: "Generic data",
					Location:    "body",
				},
			},
			expectedReqs: []string{"data"},
			description:  "Should handle generic content type like JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test tool
			tool, err := createTestTool(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to create test tool: %v", err)
			}
			if tool == nil {
				t.Fatal("Failed to get tool from test spec")
			}

			// Extract properties
			props, reqs := adapter.extractRequestBodyProperties(tool)

			// Verify properties count
			if len(props) != len(tt.expectedProps) {
				t.Errorf("Expected %d properties, got %d", len(tt.expectedProps), len(props))
			}

			// Verify each property
			for propName, expectedProp := range tt.expectedProps {
				actualProp, exists := props[propName]
				if !exists {
					t.Errorf("Expected property %s not found", propName)
					continue
				}

				if actualProp.Type != expectedProp.Type {
					t.Errorf("Property %s: expected type %s, got %s", propName, expectedProp.Type, actualProp.Type)
				}

				if actualProp.Description != expectedProp.Description {
					t.Errorf("Property %s: expected description %s, got %s", propName, expectedProp.Description, actualProp.Description)
				}

				if actualProp.Location != expectedProp.Location {
					t.Errorf("Property %s: expected location %s, got %s", propName, expectedProp.Location, actualProp.Location)
				}
			}

			// Verify required fields
			if len(reqs) != len(tt.expectedReqs) {
				t.Errorf("Expected %d required fields, got %d", len(tt.expectedReqs), len(reqs))
			}

			// Check each required field exists
			reqMap := make(map[string]bool)
			for _, req := range reqs {
				reqMap[req] = true
			}

			for _, expectedReq := range tt.expectedReqs {
				if !reqMap[expectedReq] {
					t.Errorf("Expected required field %s not found", expectedReq)
				}
			}
		})
	}
}

func TestExtractRequestBodyProperties_NoRequestBody(t *testing.T) {
	adapter := NewLibopenAPIAdapter()

	// Create operation without request body
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
`

	config := datamodel.NewDocumentConfiguration()
	document, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), config)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	docModel, errors := document.BuildV3Model()
	if len(errors) > 0 {
		t.Fatalf("Failed to build model: %v", errors[0])
	}

	// Get the GET operation
	var operation *v3.Operation
	for pathPairs := docModel.Model.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
		pathItem := pathPairs.Value()
		if pathItem.Get != nil {
			operation = pathItem.Get
			break
		}
	}

	if operation == nil {
		t.Fatal("Failed to get operation from test spec")
	}

	// Create tool wrapper
	tool := &OpenAPIMcpTool{
		Operation: operation,
		OpenAPIHandlerInput: &OpenAPIHandlerInput{
			Method:      "GET",
			Path:        "/test",
			ContentType: "application/json",
		},
	}

	// Extract properties
	props, reqs := adapter.extractRequestBodyProperties(tool)

	// Should return empty results
	if len(props) != 0 {
		t.Errorf("Expected 0 properties for no request body, got %d", len(props))
	}

	if len(reqs) != 0 {
		t.Errorf("Expected 0 required fields for no request body, got %d", len(reqs))
	}
}

func TestExtractRequestBodyProperties_EmptySchema(t *testing.T) {
	adapter := NewLibopenAPIAdapter()

	operation, err := createTestOperation(`
        required: true
        content:
          application/json:
            schema:
              type: object`)

	if err != nil {
		t.Fatalf("Failed to create test operation: %v", err)
	}
	if operation == nil {
		t.Fatal("Failed to get operation from test spec")
	}

	// Create tool wrapper
	tool := &OpenAPIMcpTool{
		Operation: operation,
		OpenAPIHandlerInput: &OpenAPIHandlerInput{
			Method:      "POST",
			Path:        "/test",
			ContentType: "application/json",
		},
	}

	// Extract properties
	props, reqs := adapter.extractRequestBodyProperties(tool)

	// Should return empty results for schema without properties
	if len(props) != 0 {
		t.Errorf("Expected 0 properties for empty schema, got %d", len(props))
	}

	if len(reqs) != 0 {
		t.Errorf("Expected 0 required fields for empty schema, got %d", len(reqs))
	}
}
