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
	contentType, _ := a.determineContentType(operation)
	var resultTool OpenAPIMcpTool = OpenAPIMcpTool{
		Operation: operation,
		OpenAPIHandlerInput: &OpenAPIHandlerInput{
			Method:      method,
			Path:        path,
			Headers:     make(map[string]string),
			Cookies:     make(map[string]string),
			BodyAppend:  make(map[string]any),
			ContentType: contentType,
		},
	}
	resultTool.Name = a.getToolName(&resultTool)
	resultTool.InputSchema = a.getToolInputSchema(&resultTool)
	resultTool.Annotations = a.getToolAnnotations(&resultTool)

	// TODO: add proper "GetToolDescription" function which handles everything related to tool description
	// Create tool description
	description := operation.Description
	if description == "" {
		description = operation.Summary
	}
	if description == "" {
		description = fmt.Sprintf("%s %s", strings.ToUpper(method), path)
	}
	// Add schema documentation for non-JSON request bodies
	bodySchemaDoc := a.extractRequestBodySchemaDoc(&resultTool)
	if bodySchemaDoc != "" {
		description = description + "\n\n" + bodySchemaDoc
	}
	// Generate samples for better AI understanding
	samples, err := a.generateToolSamples(operation, method, path)
	if err == nil && samples != "" {
		description = description + "\n\n" + samples
	}
	resultTool.Description = description
	return resultTool
}

// getToolName generates a tool name from operation ID or method and path.
func (a *LibopenAPIAdapter) getToolName(tool *OpenAPIMcpTool) string {
	toolName := tool.Operation.OperationId
	if toolName == "" {
		toolName = fmt.Sprintf("%s_%s", tool.OpenAPIHandlerInput.Method, tool.OpenAPIHandlerInput.Path)
	}

	// Clean up the tool name by removing invalid characters
	replacer := strings.NewReplacer(
		// TODO: check for other illegal chars in tool names
		"{", "",
		"}", "",
		"/", "_",
		"-", "_",
	)
	toolName = strings.ToLower(replacer.Replace(toolName))
	// TODO check tool name convention!

	return toolName
}

// getToolInputSchema creates the input schema for a tool.
func (a *LibopenAPIAdapter) getToolInputSchema(tool *OpenAPIMcpTool) core.McpToolInputSchema {
	genericProps := make(map[string]any)
	var required []string

	// Extract path, query, header, and cookie parameters
	for _, in := range ParameterLocations {
		props, reqs := a.extractParametersByIn(tool, in)
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
	bodyProps, bodyReqs := a.extractRequestBodyProperties(tool)
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
func (a *LibopenAPIAdapter) getToolAnnotations(tool *OpenAPIMcpTool) core.McpToolAnnotation {
	annotation := core.McpToolAnnotation{Title: tool.Name}
	switch methodUpper := strings.ToUpper(tool.OpenAPIHandlerInput.Method); methodUpper {
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

// GetSchemaTypeString returns the type string from a libopenapi schema
func GetSchemaTypeString(schemaProxy *base.SchemaProxy) string {
	if schemaProxy != nil {
		schema := schemaProxy.Schema()
		if schema != nil && len(schema.Type) > 0 {
			return schema.Type[0]
		}
	}
	return "string"
}

// extractParametersByIn extracts parameters of a given 'in' type from an operation
func (a *LibopenAPIAdapter) extractParametersByIn(tool *OpenAPIMcpTool, in ParameterLocation) (map[string]ToolInputProperty, []string) {
	operation := tool.Operation
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
			typeName := GetSchemaTypeString(param.Schema)
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

	for propName, propSchemaProxy := range schema.Properties.FromNewest() {
		required := ""
		if contains(schema.Required, propName) {
			required = " (required)"
		}

		doc.WriteString(fmt.Sprintf("- %s: %s%s", propName, GetSchemaTypeString(propSchemaProxy), required))
		if propSchemaProxy.Schema() != nil && propSchemaProxy.Schema().Description != "" {
			doc.WriteString(fmt.Sprintf(" - %s", propSchemaProxy.Schema().Description))
		}
		doc.WriteString("\n")
	}

	doc.WriteString(fmt.Sprintf("\nProvide the complete %s as a string in the 'body' parameter.", contentType))
	return doc.String()
}

// determineContentType returns the preferred content type for an operation's request body
func (a *LibopenAPIAdapter) determineContentType(operation *v3.Operation) (string, *v3.MediaType) {
	if operation.RequestBody == nil || operation.RequestBody.Content == nil {
		return "", nil
	}

	// Priority order for content types
	contentTypes := a.contentTypeRegistry.GetAllContentTypes()
	// TODO: check if the order of content types is correct
	for _, expectedContentType := range contentTypes {
		for actualContentType, media := range operation.RequestBody.Content.FromNewest() {
			if expectedContentType == actualContentType {
				return actualContentType, media
			}
		}
	}

	// Return first available content type if no priority match
	return operation.RequestBody.Content.First().Key(), operation.RequestBody.Content.First().Value()
}

// extractRequestBodySchemaDoc extracts schema documentation for non-JSON content types
func (a *LibopenAPIAdapter) extractRequestBodySchemaDoc(tool *OpenAPIMcpTool) string {
	if !hasRequestBody(tool.Operation) {
		return ""
	}
	// Check for non-JSON content types that need schema documentation
	nonJSONContentTypes := []string{"text/xml", "application/xml", "text/plain"}
	contentType := tool.OpenAPIHandlerInput.ContentType
	if slices.Contains(nonJSONContentTypes, contentType) {
		content, ok := tool.Operation.RequestBody.Content.Get(contentType)
		if ok {
			return a.generateSchemaDocumentation(content.Schema, contentType)
		}
	}
	return "" // in case no content type matches
}

func hasRequestBody(operation *v3.Operation) bool {
	return operation.RequestBody != nil && operation.RequestBody.Content != nil
}

// extractRequestBodyProperties extracts properties from the request body schema
func (a *LibopenAPIAdapter) extractRequestBodyProperties(tool *OpenAPIMcpTool) (map[string]ToolInputProperty, []string) {
	if !hasRequestBody(tool.Operation) {
		return make(map[string]ToolInputProperty), []string{}
	}
	contentType := tool.OpenAPIHandlerInput.ContentType
	contentMedia, ok := tool.Operation.RequestBody.Content.Get(contentType)
	if !ok {
		// If no recognized content type found, try the first available one
		contentPairs := tool.Operation.RequestBody.Content.First()
		return a.extractPropertiesFromMedia(contentPairs.Value(), contentPairs.Key())
	}
	return a.extractPropertiesFromMedia(contentMedia, contentType)
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
	for contentType, mediaType := range operation.RequestBody.Content.FromNewest() {
		if mediaType.Schema == nil {
			continue
		}
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
				for contentType, mediaType := range response.Content.FromNewest() {
					if mediaType.Schema == nil {
						continue
					}
					mockGen := renderer.NewMockGenerator(renderer.JSON)
					mockGen.SetPretty()
					mockGen.DisableRequiredCheck() // Show all properties, including system-generated fields
					sample, err := mockGen.GenerateMock(mediaType.Schema.Schema(), "")
					if err != nil {
						return err
					}
					fmt.Fprintf(samples, "Content-Type: %s\n", contentType)
					fmt.Fprintf(samples, "```json\n%s\n```\n\n", string(sample)) // TODO: json is wrong here, should be contentType
					break
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
