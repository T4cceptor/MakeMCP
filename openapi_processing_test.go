package main

import (
	"encoding/json"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestLoadOpenAPISpec(t *testing.T) {
	// Test only the valid case since loadOpenAPISpec calls log.Fatalf on error
	// which would exit the test process
	t.Run("Valid OpenAPI spec file", func(t *testing.T) {
		doc := loadOpenAPISpec("testdata/sample_openapi.json")
		
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
		if doc.Info.Title != "Sample User API" {
			t.Errorf("Expected title 'Sample User API', got %s", doc.Info.Title)
		}
		if doc.Info.Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got %s", doc.Info.Version)
		}
	})
}

func TestFromOpenAPISpecs(t *testing.T) {
	params := CLIParams{
		Specs:   "testdata/sample_openapi.json",
		BaseURL: "https://api.example.com",
		DevMode: true, // Suppress security warnings for tests
	}

	app := FromOpenAPISpecs(params)

	// Test basic app structure
	if app.Name == "" {
		t.Error("Expected non-empty app name")
	}
	if app.Version == "" {
		t.Error("Expected non-empty app version")
	}
	if app.OpenAPIConfig == nil {
		t.Error("Expected OpenAPIConfig but got nil")
	}
	if app.OpenAPIConfig.BaseUrl != params.BaseURL {
		t.Errorf("Expected BaseUrl %s, got %s", params.BaseURL, app.OpenAPIConfig.BaseUrl)
	}

	// Test tools generation
	expectedTools := []string{
		"listUsers",
		"createUser", 
		"getUserById",
		"updateUser",
		"deleteUser",
		"GET_/users/{userId}/preferences", // This one doesn't have operationId
	}

	if len(app.Tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(app.Tools))
	}

	// Check that all expected tools are present
	toolNames := make(map[string]bool)
	for _, tool := range app.Tools {
		toolNames[tool.Name] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolNames[expectedTool] {
			t.Errorf("Expected tool %s not found", expectedTool)
		}
	}
}

func TestGetToolName(t *testing.T) {
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
			expected:    "listUsers",
		},
		{
			name:        "Without operation ID",
			method:      "POST",
			path:        "/users/{id}",
			operationID: "",
			expected:    "POST_/users/{id}",
		},
		{
			name:        "DELETE with operation ID",
			method:      "DELETE",
			path:        "/users/{id}",
			operationID: "deleteUser",
			expected:    "deleteUser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operation := &openapi3.Operation{
				OperationID: tt.operationID,
			}
			
			result := GetToolName(tt.method, tt.path, operation)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetToolInputSchema(t *testing.T) {
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

	schema := GetToolInputSchema("PUT", "/users/{userId}", operation)

	// Check schema structure
	if schema.Type != "object" {
		t.Errorf("Expected schema type 'object', got %s", schema.Type)
	}

	// Check expected properties with prefixes
	expectedProperties := map[string]string{
		"path__userId":    "string",
		"query__limit":    "integer",
		"header__X-API-Key": "string",
		"body__name":      "string",
		"body__email":     "string",
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

func TestGetToolAnnotations(t *testing.T) {
	operation := &openapi3.Operation{
		Summary:     "Test operation",
		Description: "This is a test operation",
	}

	tests := []struct {
		name           string
		method         string
		path           string
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
			expectedIdempotent:  nil,
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
			expectedIdempotent:  boolPtr(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := GetToolAnnotations(tt.method, tt.path, operation)
			
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

func TestGetToolDescription(t *testing.T) {
	operation := &openapi3.Operation{
		Summary:     "List users",
		Description: "Retrieve a paginated list of users with optional filtering",
	}

	// Create a simple tool input schema
	toolInputSchema := mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"path__userId": map[string]interface{}{
				"type":        "string",
				"description": "User ID",
			},
			"query__limit": map[string]interface{}{
				"type":        "integer",
				"description": "Limit results",
			},
			"header__X-API-Key": map[string]interface{}{
				"type":        "string",
				"description": "API Key",
			},
		},
		Required: []string{"path__userId", "header__X-API-Key"},
	}

	description := GetToolDescription("GET", "/users/{userId}", operation, toolInputSchema)
	
	// Check that description contains expected elements
	if description == "" {
		t.Error("Expected non-empty description")
	}
	
	// Should contain operation summary
	if !contains(description, "List users") {
		t.Error("Expected description to contain operation summary")
	}
	
	// Should contain operation description
	if !contains(description, "paginated list of users") {
		t.Error("Expected description to contain operation description")
	}
	
	// Should contain parameter sections
	if !contains(description, "Path Parameters:") {
		t.Error("Expected description to contain Path Parameters section")
	}
	
	if !contains(description, "Query Parameters:") {
		t.Error("Expected description to contain Query Parameters section")
	}
	
	if !contains(description, "Header Parameters:") {
		t.Error("Expected description to contain Header Parameters section")
	}
	
	// Should contain example input
	if !contains(description, "Example input:") {
		t.Error("Expected description to contain Example input section")
	}
	
	// Should contain prefix instruction
	if !contains(description, "prefix format") {
		t.Error("Expected description to contain prefix format instruction")
	}
}

func TestParsePrefixedParameters(t *testing.T) {
	input := map[string]any{
		"path__userId":     "user123",
		"query__limit":     10,
		"header__X-API-Key": "secret-key",
		"body__name":       "John Doe",
		"body__email":      "john@example.com",
		"invalid_param":    "should be ignored",
	}

	params := parsePrefixedParameters(input)

	// Check path parameters
	if params.Path["userId"] != "user123" {
		t.Errorf("Expected path userId 'user123', got %v", params.Path["userId"])
	}

	// Check query parameters
	if params.Query["limit"] != 10 {
		t.Errorf("Expected query limit 10, got %v", params.Query["limit"])
	}

	// Check header parameters
	if params.Header["X-API-Key"] != "secret-key" {
		t.Errorf("Expected header X-API-Key 'secret-key', got %v", params.Header["X-API-Key"])
	}

	// Check body parameters
	if params.Body["name"] != "John Doe" {
		t.Errorf("Expected body name 'John Doe', got %v", params.Body["name"])
	}
	if params.Body["email"] != "john@example.com" {
		t.Errorf("Expected body email 'john@example.com', got %v", params.Body["email"])
	}

	// Check that invalid parameter was ignored
	if len(params.Path) != 1 || len(params.Query) != 1 || len(params.Header) != 1 || len(params.Body) != 2 {
		t.Error("Invalid parameter should have been ignored")
	}
}

func TestGenerateExampleInput(t *testing.T) {
	params := []ParameterInfo{
		{Name: "userId", Type: "string", Location: "path", Required: true},
		{Name: "limit", Type: "integer", Location: "query", Required: false},
		{Name: "email", Type: "string", Location: "body", Required: true},
		{Name: "active", Type: "boolean", Location: "body", Required: false},
	}

	exampleJSON := generateExampleInput(params)
	
	// Parse the generated JSON
	var example map[string]any
	if err := json.Unmarshal([]byte(exampleJSON), &example); err != nil {
		t.Fatalf("Failed to parse example JSON: %v", err)
	}

	// Check that all parameters are present with correct prefixes
	expectedKeys := []string{"path__userId", "query__limit", "body__email", "body__active"}
	for _, key := range expectedKeys {
		if _, exists := example[key]; !exists {
			t.Errorf("Expected key %s not found in example", key)
		}
	}

	// Check types
	if example["path__userId"] != "example string" {
		t.Errorf("Expected path__userId to be 'example string', got %v", example["path__userId"])
	}
	if example["query__limit"] != float64(42) { // JSON unmarshals numbers as float64
		t.Errorf("Expected query__limit to be 42, got %v", example["query__limit"])
	}
	if example["body__email"] != "user@example.com" {
		t.Errorf("Expected body__email to be 'user@example.com', got %v", example["body__email"])
	}
	if example["body__active"] != true {
		t.Errorf("Expected body__active to be true, got %v", example["body__active"])
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}