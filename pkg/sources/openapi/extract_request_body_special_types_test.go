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
func createTestOperationSpecialTypes(requestBody string) (*v3.Operation, error) {
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
func createTestToolSpecialTypes(requestBody string) (*OpenAPIMcpTool, error) {
	operation, err := createTestOperationSpecialTypes(requestBody)
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

func TestExtractRequestBodyProperties_FormData(t *testing.T) {
	adapter := NewLibopenAPIAdapter()

	tests := []struct {
		name          string
		requestBody   string
		expectedProps map[string]ToolInputProperty
		expectedReqs  []string
		description   string
	}{
		{
			name: "application/x-www-form-urlencoded with structured schema",
			requestBody: `
        required: true
        content:
          application/x-www-form-urlencoded:
            schema:
              type: object
              properties:
                username:
                  type: string
                  description: User login name
                password:
                  type: string
                  description: User password
                remember:
                  type: boolean
                  description: Remember login
              required: [username, password]`,
			expectedProps: map[string]ToolInputProperty{
				"form__username": {
					Type:        "string",
					Description: "User login name",
					Location:    "body",
				},
				"form__password": {
					Type:        "string",
					Description: "User password",
					Location:    "body",
				},
				"form__remember": {
					Type:        "boolean",
					Description: "Remember login",
					Location:    "body",
				},
			},
			expectedReqs: []string{"form__username", "form__password"},
			description:  "Should create form__ prefixed parameters for structured form data",
		},
		{
			name: "form-urlencoded without schema (falls back to body)",
			requestBody: `
        required: true
        content:
          application/x-www-form-urlencoded:
            schema:
              type: string
              description: Raw form data`,
			expectedProps: map[string]ToolInputProperty{
				"body": {
					Type:        "string",
					Description: "Form URL-encoded request body. ",
					Location:    "body",
				},
			},
			expectedReqs: []string{"body"},
			description:  "Should fall back to single body parameter when schema is not structured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test tool
			tool, err := createTestToolSpecialTypes(tt.requestBody)
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
				t.Logf("Got properties: %+v", props)
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
				t.Logf("Got required: %v, Expected: %v", reqs, tt.expectedReqs)
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

func TestExtractRequestBodyProperties_Multipart(t *testing.T) {
	adapter := NewLibopenAPIAdapter()

	tests := []struct {
		name          string
		requestBody   string
		expectedProps map[string]ToolInputProperty
		expectedReqs  []string
		description   string
	}{
		{
			name: "multipart/form-data with file upload",
			requestBody: `
        required: true
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                file:
                  type: string
                  format: binary
                  description: File to upload
                name:
                  type: string
                  description: File name
                category:
                  type: string
                  description: File category
              required: [file, name]`,
			expectedProps: map[string]ToolInputProperty{
				"multipart__file": {
					Type:        "file",
					Description: "File to upload",
					Location:    "body",
				},
				"multipart__name": {
					Type:        "string",
					Description: "File name",
					Location:    "body",
				},
				"multipart__category": {
					Type:        "string",
					Description: "File category",
					Location:    "body",
				},
			},
			expectedReqs: []string{"multipart__file", "multipart__name"},
			description:  "Should create multipart__ prefixed parameters with file type detection",
		},
		{
			name: "multipart without structured schema (falls back to body)",
			requestBody: `
        required: true
        content:
          multipart/form-data:
            schema:
              type: string
              description: Raw multipart data`,
			expectedProps: map[string]ToolInputProperty{
				"body": {
					Type:        "string",
					Description: "Multipart form data request body",
					Location:    "body",
				},
			},
			expectedReqs: []string{"body"},
			description:  "Should fall back to single body parameter when schema is not structured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test tool
			tool, err := createTestToolSpecialTypes(tt.requestBody)
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
				t.Logf("Got properties: %+v", props)
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
				t.Logf("Got required: %v, Expected: %v", reqs, tt.expectedReqs)
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

func TestExtractRequestBodyProperties_StructuredXML(t *testing.T) {
	adapter := NewLibopenAPIAdapter()

	tests := []struct {
		name          string
		requestBody   string
		expectedProps map[string]ToolInputProperty
		expectedReqs  []string
		description   string
	}{
		{
			name: "text/xml with structured schema (no prefix)",
			requestBody: `
        required: true
        content:
          text/xml:
            schema:
              type: object
              properties:
                CompleteMultipartUpload:
                  type: object
                  description: The container for the completed multipart upload details.
                  properties:
                    Parts:
                      type: array
                      description: Array of CompletedPart data types.
              required: [CompleteMultipartUpload]`,
			expectedProps: map[string]ToolInputProperty{
				"CompleteMultipartUpload": {
					Type:        "object",
					Description: "The container for the completed multipart upload details.",
					Location:    "body",
				},
			},
			expectedReqs: []string{"CompleteMultipartUpload"},
			description:  "Should extract structured XML properties without prefix (like JSON)",
		},
		{
			name: "text/xml without structured schema (falls back to body)",
			requestBody: `
        required: true
        content:
          text/xml:
            schema:
              type: string
              description: Raw XML content`,
			expectedProps: map[string]ToolInputProperty{
				"body": {
					Type:        "string",
					Description: "XML request body content",
					Location:    "body",
				},
			},
			expectedReqs: []string{"body"},
			description:  "Should fall back to single body parameter for non-structured XML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test tool
			tool, err := createTestToolSpecialTypes(tt.requestBody)
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
				t.Logf("Got properties: %+v", props)
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
				t.Logf("Got required: %v, Expected: %v", reqs, tt.expectedReqs)
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
