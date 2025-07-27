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

// Helper function to create AWS S3 style test operation with $refs
func createTestOperationWithRefs(requestBody string) (*v3.Operation, error) {
	// Create an OpenAPI spec that mimics the AWS S3 structure with $refs
	spec := `
openapi: 3.0.0
info:
  title: Test API with Refs
  version: 1.0.0
components:
  schemas:
    CompletedPart:
      type: object
      properties:
        ETag:
          type: string
          description: Entity tag returned when the part was uploaded.
        PartNumber:
          type: integer
          description: Part number that was specified in the UploadPart request.
        ChecksumCRC32:
          type: string
          description: The base64-encoded, 32-bit CRC32 checksum of the object.
      required: [ETag, PartNumber]
      
    CompletedPartList:
      type: array
      items:
        $ref: "#/components/schemas/CompletedPart"
      description: Array of CompletedPart data types.
      
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
	config.AllowFileReferences = true
	config.AllowRemoteReferences = true

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

func TestExtractRequestBodyProperties_WithRefs(t *testing.T) {
	adapter := NewLibopenAPIAdapter()

	tests := []struct {
		name          string
		requestBody   string
		expectedProps map[string]ToolInputProperty
		expectedReqs  []string
		description   string
	}{
		{
			name: "AWS S3 CompleteMultipartUpload with $refs",
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
                      allOf:
                        - $ref: "#/components/schemas/CompletedPartList"
                        - xml:
                            name: Part
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
			description:  "Should extract structured properties even with complex $refs and allOf",
		},
		{
			name: "Direct reference to array schema",
			requestBody: `
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                parts:
                  $ref: "#/components/schemas/CompletedPartList"
                metadata:
                  type: string
                  description: Additional metadata
              required: [parts]`,
			expectedProps: map[string]ToolInputProperty{
				"parts": {
					Type:        "array",
					Description: "Array of CompletedPart data types.",
					Location:    "body",
				},
				"metadata": {
					Type:        "string",
					Description: "Additional metadata",
					Location:    "body",
				},
			},
			expectedReqs: []string{"parts"},
			description:  "Should resolve direct $ref to array schema",
		},
		{
			name: "Direct reference to object schema",
			requestBody: `
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                part:
                  $ref: "#/components/schemas/CompletedPart"
                count:
                  type: integer
                  description: Number of parts
              required: [part]`,
			expectedProps: map[string]ToolInputProperty{
				"part": {
					Type:        "object",
					Description: "",
					Location:    "body",
				},
				"count": {
					Type:        "integer",
					Description: "Number of parts",
					Location:    "body",
				},
			},
			expectedReqs: []string{"part"},
			description:  "Should resolve direct $ref to object schema",
		},
		{
			name: "Nested object with mixed refs and properties",
			requestBody: `
        required: true
        content:
          application/xml:
            schema:
              type: object
              properties:
                upload:
                  type: object
                  description: Upload container
                  properties:
                    parts:
                      $ref: "#/components/schemas/CompletedPartList"
                    uploadId:
                      type: string
                      description: Upload identifier
                name:
                  type: string
                  description: Object name
              required: [upload, name]`,
			expectedProps: map[string]ToolInputProperty{
				"upload": {
					Type:        "object",
					Description: "Upload container",
					Location:    "body",
				},
				"name": {
					Type:        "string",
					Description: "Object name",
					Location:    "body",
				},
			},
			expectedReqs: []string{"upload", "name"},
			description:  "Should handle nested objects with mixed $refs and direct properties",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test operation
			operation, err := createTestOperationWithRefs(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to create test operation: %v", err)
			}
			if operation == nil {
				t.Fatal("Failed to get operation from test spec")
			}
			tool := OpenAPIMcpTool{
				Operation: operation,
				OpenAPIHandlerInput: &OpenAPIHandlerInput{
					ContentType: "*/*",
				},
			}

			// Extract properties
			props, reqs := adapter.extractRequestBodyProperties(&tool)

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
