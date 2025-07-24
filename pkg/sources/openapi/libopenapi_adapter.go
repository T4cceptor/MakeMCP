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
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/renderer"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
)

// LibopenAPIAdapter contains all libopenapi-specific functionality
// This isolates the library-specific code and makes it easier to swap libraries later
type LibopenAPIAdapter struct {
	contentTypeRegistry *ContentTypeRegistry
}

// NewLibopenAPIAdapter creates a new adapter instance
func NewLibopenAPIAdapter() *LibopenAPIAdapter {
	return &LibopenAPIAdapter{
		contentTypeRegistry: NewContentTypeRegistry(),
	}
}

// LoadOpenAPISpec loads an OpenAPI specification using libopenapi
func (a *LibopenAPIAdapter) LoadOpenAPISpec(openAPISpecLocation string, strictValidation bool) (*libopenapi.DocumentModel[v3.Document], error) {
	log.Println("Loading OpenAPI spec from:", openAPISpecLocation)

	// Load specification bytes
	specBytes, err := a.loadSpecBytes(openAPISpecLocation)
	if err != nil {
		return nil, err
	}

	// Create document with configuration
	config := datamodel.NewDocumentConfiguration()
	config.AllowFileReferences = true
	config.AllowRemoteReferences = true

	document, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	// Build V3 model
	docModel, errors := document.BuildV3Model()
	if len(errors) > 0 {
		if strictValidation {
			var errorMessages []string
			for _, err := range errors {
				errorMessages = append(errorMessages, err.Error())
			}
			return nil, fmt.Errorf("OpenAPI model validation errors: %s", strings.Join(errorMessages, "; "))
		} else {
			log.Printf("OpenAPI validation warnings (permissive mode): %d warnings", len(errors))
		}
	}

	log.Printf("Loaded OpenAPI spec: %s v%s", docModel.Model.Info.Title, docModel.Model.Info.Version)
	return docModel, nil
}

// loadSpecBytes loads specification bytes from either a file or URL
func (a *LibopenAPIAdapter) loadSpecBytes(openAPISpecLocation string) ([]byte, error) {
	sourceType := a.detectSourceType(openAPISpecLocation)

	switch sourceType {
	case "file":
		specBytes, err := os.ReadFile(openAPISpecLocation)
		if err != nil {
			return nil, fmt.Errorf("failed to read OpenAPI spec file: %w", err)
		}
		return specBytes, nil
	case "url":
		resp, err := http.Get(openAPISpecLocation)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch OpenAPI spec from URL: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("failed to close response body: %v", err)
			}
		}()
		specBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read OpenAPI spec response: %w", err)
		}
		return specBytes, nil
	default:
		return nil, fmt.Errorf("unknown source type: %s", sourceType)
	}
}

// detectSourceType determines whether the spec location is a URL or file path
func (a *LibopenAPIAdapter) detectSourceType(openAPISpecLocation string) string {
	if strings.HasPrefix(openAPISpecLocation, "http://") || strings.HasPrefix(openAPISpecLocation, "https://") {
		return "url"
	}
	return "file"
}

// ForEachOperation iterates through all operations in the OpenAPI document
func (a *LibopenAPIAdapter) ForEachOperation(doc *libopenapi.DocumentModel[v3.Document], callback func(method, path string, operation *v3.Operation) error) error {
	for pathPairs := doc.Model.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
		path := pathPairs.Key()
		pathItem := pathPairs.Value()

		operations := pathItem.GetOperations()
		for opPairs := operations.First(); opPairs != nil; opPairs = opPairs.Next() {
			method := opPairs.Key()
			operation := opPairs.Value()

			if err := callback(method, path, operation); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetDocumentInfo extracts basic document information
func (a *LibopenAPIAdapter) GetDocumentInfo(doc *libopenapi.DocumentModel[v3.Document]) (title, version string) {
	return doc.Model.Info.Title, doc.Model.Info.Version
}

// CreateToolsFromDocument converts all OpenAPI operations in a document to MCP tools
func (a *LibopenAPIAdapter) CreateToolsFromDocument(doc *libopenapi.DocumentModel[v3.Document]) ([]OpenAPIMcpTool, error) {
	var tools []OpenAPIMcpTool

	err := a.ForEachOperation(doc, func(method, path string, operation *v3.Operation) error {
		tool := a.createToolFromOperation(method, path, operation)
		tools = append(tools, tool)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tools, nil
}

// createToolFromOperation creates a MakeMCPTool from an OpenAPI operation
func (a *LibopenAPIAdapter) createToolFromOperation(method, path string, operation *v3.Operation) OpenAPIMcpTool {
	toolName := a.getToolName(method, path, operation)
	toolInputSchema := a.getToolInputSchema(operation)
	toolAnnotations := a.getToolAnnotations(method, path, operation)

	// Create tool description
	description := operation.Description
	if description == "" {
		description = operation.Summary
	}
	if description == "" {
		description = fmt.Sprintf("%s %s", strings.ToUpper(method), path)
	}

	// Add schema documentation for non-JSON request bodies
	bodySchemaDoc := a.extractRequestBodySchemaDoc(operation)
	if bodySchemaDoc != "" {
		description = description + "\n\n" + bodySchemaDoc
	}

	// Generate samples for better AI understanding
	samples, err := a.generateToolSamples(operation, method, path)
	if err == nil && samples != "" {
		description = description + "\n\n" + samples
	}

	// Create the tool
	tool := OpenAPIMcpTool{
		McpTool: core.McpTool{
			Name:        toolName,
			Description: description,
			InputSchema: toolInputSchema,
			Annotations: toolAnnotations,
		},
		OpenAPIHandlerInput: &OpenAPIHandlerInput{
			Method:      method,
			Path:        path,
			Headers:     make(map[string]string),
			Cookies:     make(map[string]string),
			BodyAppend:  make(map[string]any),
			ContentType: a.determineContentType(operation),
		},
	}

	return tool
}

// getToolName generates a tool name from operation ID or method and path.
func (a *LibopenAPIAdapter) getToolName(method string, path string, operation *v3.Operation) string {
	toolName := operation.OperationId
	if toolName == "" {
		toolName = fmt.Sprintf("%s_%s", method, path)
	}

	// Clean up the tool name by removing invalid characters
	replacer := strings.NewReplacer(
		"{", "",
		"}", "",
		"/", "_",
		"-", "_",
	)
	toolName = strings.ToLower(replacer.Replace(toolName))

	return toolName
}

// getToolInputSchema creates the input schema for a tool.
func (a *LibopenAPIAdapter) getToolInputSchema(operation *v3.Operation) core.McpToolInputSchema {
	genericProps := make(map[string]any)
	var required []string

	// Extract path, query, header, and cookie parameters
	for _, in := range []ParameterLocation{ParameterLocationPath, ParameterLocationQuery, ParameterLocationHeader, ParameterLocationCookie} {
		props, reqs := a.extractParametersByIn(operation, in)
		for paramName, prop := range props {
			prefixedName := fmt.Sprintf("%s__%s", in, paramName)
			genericProps[prefixedName] = map[string]any{
				"type":        prop.Type,
				"description": prop.Description,
			}
			if slices.Contains(reqs, paramName) {
				required = append(required, prefixedName)
			}
		}
	}

	// Extract request body properties
	bodyProps, bodyReqs := a.extractRequestBodyProperties(operation)
	for paramName, prop := range bodyProps {
		prefixedName := fmt.Sprintf("body__%s", paramName)
		genericProps[prefixedName] = map[string]any{
			"type":        prop.Type,
			"description": prop.Description,
		}
		if slices.Contains(bodyReqs, paramName) {
			required = append(required, prefixedName)
		}
	}

	return core.McpToolInputSchema{
		Type:       "object",
		Properties: genericProps,
		Required:   required,
	}
}

// getToolAnnotations returns tool annotations based on HTTP method and operation.
func (a *LibopenAPIAdapter) getToolAnnotations(method string, path string, operation *v3.Operation) core.McpToolAnnotation {
	annotation := core.McpToolAnnotation{
		Title:           a.getToolName(method, path, operation),
		ReadOnlyHint:    nil,
		DestructiveHint: nil,
		IdempotentHint:  nil,
		OpenWorldHint:   nil,
	}

	switch methodUpper := strings.ToUpper(method); methodUpper {
	case "GET", "HEAD", "OPTIONS":
		// ReadOnlyHint: GET, HEAD, OPTIONS are considered read-only and idempotent
		annotation.ReadOnlyHint = boolPtr(true)
		annotation.IdempotentHint = boolPtr(true)
	case "DELETE":
		// DestructiveHint: DELETE is considered destructive
		annotation.DestructiveHint = boolPtr(true)
	case "PUT":
		// IdempotentHint: PUT is idempotent
		annotation.IdempotentHint = boolPtr(true)
	case "POST":
		// IdempotentHint: POST is not idempotent
		annotation.IdempotentHint = boolPtr(false)
	}

	return annotation
}

// getSchemaTypeString returns the type string from a libopenapi schema
func (a *LibopenAPIAdapter) getSchemaTypeString(schemaProxy *base.SchemaProxy) string {
	if schemaProxy != nil {
		schema := schemaProxy.Schema()
		if schema != nil && len(schema.Type) > 0 {
			return schema.Type[0]
		}
	}
	return "string"
}

// extractParametersByIn extracts parameters of a given 'in' type from an operation
func (a *LibopenAPIAdapter) extractParametersByIn(operation *v3.Operation, in ParameterLocation) (map[string]ToolInputProperty, []string) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	if operation.Parameters == nil {
		return properties, required
	}

	for _, param := range operation.Parameters {
		if param == nil {
			continue
		}
		if param.In == string(in) {
			typeName := a.getSchemaTypeString(param.Schema)
			properties[param.Name] = ToolInputProperty{
				Type:        typeName,
				Description: param.Description,
				Location:    in,
			}
			if param.Required != nil && *param.Required {
				required = append(required, param.Name)
			}
		}
	}
	return properties, required
}

// generateSchemaDocumentation creates human-readable schema documentation
func (a *LibopenAPIAdapter) generateSchemaDocumentation(schemaProxy *base.SchemaProxy, contentType string) string {
	if schemaProxy == nil {
		return fmt.Sprintf("Provide %s content as a string.", contentType)
	}

	schema := schemaProxy.Schema()
	if schema == nil || schema.Properties == nil {
		return fmt.Sprintf("Provide %s content as a string.", contentType)
	}

	var doc strings.Builder
	doc.WriteString(fmt.Sprintf("Expected %s structure:\n", contentType))

	for propPairs := schema.Properties.First(); propPairs != nil; propPairs = propPairs.Next() {
		propName := propPairs.Key()
		propSchemaProxy := propPairs.Value()

		required := ""
		if contains(schema.Required, propName) {
			required = " (required)"
		}

		doc.WriteString(fmt.Sprintf("- %s: %s%s", propName, a.getSchemaTypeString(propSchemaProxy), required))
		if propSchemaProxy.Schema() != nil && propSchemaProxy.Schema().Description != "" {
			doc.WriteString(fmt.Sprintf(" - %s", propSchemaProxy.Schema().Description))
		}
		doc.WriteString("\n")
	}

	doc.WriteString(fmt.Sprintf("\nProvide the complete %s as a string in the 'body' parameter.", contentType))
	return doc.String()
}

// determineContentType returns the preferred content type for an operation's request body
func (a *LibopenAPIAdapter) determineContentType(operation *v3.Operation) string {
	if operation.RequestBody == nil {
		return ""
	}

	// Priority order for content types
	contentTypes := []string{"application/json", "*/*", "text/xml", "application/xml", "text/plain"}

	for _, contentType := range contentTypes {
		if operation.RequestBody.Content != nil {
			for contentPairs := operation.RequestBody.Content.First(); contentPairs != nil; contentPairs = contentPairs.Next() {
				if contentPairs.Key() == contentType {
					return contentType
				}
			}
		}
	}

	// Return first available content type if no priority match
	if operation.RequestBody.Content != nil {
		for contentPairs := operation.RequestBody.Content.First(); contentPairs != nil; contentPairs = contentPairs.Next() {
			return contentPairs.Key()
		}
	}

	return ""
}

// extractRequestBodySchemaDoc extracts schema documentation for non-JSON content types
func (a *LibopenAPIAdapter) extractRequestBodySchemaDoc(operation *v3.Operation) string {
	if operation.RequestBody == nil {
		return ""
	}

	// Check for non-JSON content types that need schema documentation
	nonJSONContentTypes := []string{"text/xml", "application/xml", "text/plain"}

	for _, contentType := range nonJSONContentTypes {
		if operation.RequestBody.Content != nil {
			for contentPairs := operation.RequestBody.Content.First(); contentPairs != nil; contentPairs = contentPairs.Next() {
				if contentPairs.Key() == contentType {
					media := contentPairs.Value()
					if media.Schema != nil {
						return a.generateSchemaDocumentation(media.Schema, contentType)
					}
				}
			}
		}
	}

	return ""
}

// extractRequestBodyProperties extracts properties from the request body schema
func (a *LibopenAPIAdapter) extractRequestBodyProperties(operation *v3.Operation) (map[string]ToolInputProperty, []string) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	if operation.RequestBody == nil {
		return properties, required
	}

	// Check for supported content types in priority order
	contentTypes := []string{"application/json", "*/*", "text/xml", "application/xml"}

	for _, contentType := range contentTypes {
		if operation.RequestBody.Content != nil {
			for contentPairs := operation.RequestBody.Content.First(); contentPairs != nil; contentPairs = contentPairs.Next() {
				if contentPairs.Key() == contentType {
					return a.extractPropertiesFromMedia(contentPairs.Value(), contentType)
				}
			}
		}
	}

	// If no recognized content type found, try the first available one
	if operation.RequestBody.Content != nil {
		for contentPairs := operation.RequestBody.Content.First(); contentPairs != nil; contentPairs = contentPairs.Next() {
			return a.extractPropertiesFromMedia(contentPairs.Value(), contentPairs.Key())
		}
	}

	return properties, required
}

// extractPropertiesFromMedia extracts properties from a media type using content-type specific handlers
func (a *LibopenAPIAdapter) extractPropertiesFromMedia(media *v3.MediaType, contentType string) (map[string]ToolInputProperty, []string) {
	handler := a.contentTypeRegistry.GetHandler(contentType)

	properties, required, err := handler.ExtractParameters(media)
	if err != nil {
		// Log error and fall back to empty properties
		// In the future, we might want to handle this differently
		log.Printf("Error extracting parameters for content type %s: %v", contentType, err)
		return make(map[string]ToolInputProperty), []string{}
	}

	return properties, required
}

func generateSampleRequest(samples *strings.Builder, operation *v3.Operation) error {
	samples.WriteString("Sample Request:\n")
	for contentPairs := operation.RequestBody.Content.First(); contentPairs != nil; contentPairs = contentPairs.Next() {
		contentType := contentPairs.Key()
		mediaType := contentPairs.Value()
		if mediaType.Schema != nil {
			mockGen := renderer.NewMockGenerator(renderer.JSON)
			mockGen.SetPretty()
			mockGen.DisableRequiredCheck() // Show all properties, not just required ones
			sample, err := mockGen.GenerateMock(mediaType.Schema.Schema(), "")
			if err == nil {
				fmt.Fprintf(samples, "Content-Type: %s\n", contentType)
				fmt.Fprintf(samples, "```json\n%s\n```\n\n", string(sample))
				break // Only show one sample request
			} else {
				return err
			}
		}
	}
	return nil
}

func generateSampleResponse(samples *strings.Builder, operation *v3.Operation) error {
	for statusPairs := operation.Responses.Codes.First(); statusPairs != nil; statusPairs = statusPairs.Next() {
		statusCodeStr := statusPairs.Key()
		// Parse status code from string to int for comparison
		var statusCode int
		if _, err := fmt.Sscanf(statusCodeStr, "%d", &statusCode); err == nil && statusCode >= 200 && statusCode < 300 {
			response := statusPairs.Value()
			if response != nil && response.Content != nil {
				fmt.Fprintf(samples, "Sample Response (%d):\n", statusCode)
				for contentPairs := response.Content.First(); contentPairs != nil; contentPairs = contentPairs.Next() {
					contentType := contentPairs.Key()
					mediaType := contentPairs.Value()

					if mediaType.Schema != nil {
						mockGen := renderer.NewMockGenerator(renderer.JSON)
						mockGen.SetPretty()
						mockGen.DisableRequiredCheck() // Show all properties, including system-generated fields
						sample, err := mockGen.GenerateMock(mediaType.Schema.Schema(), "")
						if err == nil {
							fmt.Fprintf(samples, "Content-Type: %s\n", contentType)
							fmt.Fprintf(samples, "```json\n%s\n```\n\n", string(sample))
							break // Only show one sample response
						} else {
							return err
						}
					}
				}
				break // Only show first success response
			} else {
				return err
			}
		}
	}
	return nil
}

// generateToolSamples creates sample request/response examples for tool descriptions
func (a *LibopenAPIAdapter) generateToolSamples(operation *v3.Operation, method, path string) (string, error) {
	_ = method // Future use for method-specific sample generation
	_ = path   // Future use for path-specific sample generation
	var samples strings.Builder

	// Generate sample request
	if operation.RequestBody != nil {
		err := generateSampleRequest(&samples, operation)
		if err != nil {
			return "", err
		}
	}

	// Generate sample response
	if operation.Responses != nil && operation.Responses.Codes != nil {
		err := generateSampleResponse(&samples, operation)
		if err != nil {
			return "", err
		}
	}

	return samples.String(), nil
}
