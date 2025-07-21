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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
)

// OpenAPISource implements the sources.Source interface for OpenAPI specifications
type OpenAPISource struct{}

// Name returns the name of this source type
func (s *OpenAPISource) Name() string {
	return "openapi"
}

// ParseParams converts raw CLI input into typed OpenAPI parameters
func (s *OpenAPISource) ParseParams(input *core.CLIParamsInput) (core.SourceParams, error) {
	return ParseFromCLIInput(input)
}

// Parse converts an OpenAPI specification into a MakeMCPApp configuration
func (s *OpenAPISource) Parse(params core.SourceParams) (*core.MakeMCPApp, error) {
	// Type assert to OpenAPIParams
	openAPIParams, ok := params.(*OpenAPIParams)
	if !ok {
		return nil, fmt.Errorf("expected OpenAPIParams, got %T", params)
	}

	// Security checks
	if !openAPIParams.DevMode {
		WarnURLSecurity(openAPIParams.Specs, "OpenAPI spec", false)
		WarnURLSecurity(openAPIParams.BaseURL, "Base URL", false)
	}

	app := core.NewMakeMCPApp("", "", openAPIParams)

	// Load the OpenAPI specification
	doc, err := s.loadOpenAPISpec(openAPIParams.Specs, openAPIParams.StrictValidate)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	// Create the app configuration
	if app.Name == "" {
		app.Name = doc.Info.Title
	}
	if app.Version == "" {
		app.Version = doc.Info.Version
	}

	// Convert OpenAPI operations to MCP tools
	var tools []core.MakeMCPTool
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			tool := s.createToolFromOperation(method, path, operation)
			tools = append(tools, &tool)
		}
	}

	app.Tools = tools
	return &app, nil
}

// loadOpenAPISpec loads an OpenAPI specification from a URL or local file path
func (s *OpenAPISource) loadOpenAPISpec(openAPISpecLocation string, strictValidation bool) (*openapi3.T, error) {
	log.Println("Loading OpenAPI spec from:", openAPISpecLocation)
	loader := openapi3.NewLoader()
	sourceType := s.detectSourceType(openAPISpecLocation)

	var doc *openapi3.T
	var err error

	switch sourceType {
	case "file":
		doc, err = loader.LoadFromFile(openAPISpecLocation)
		if err != nil {
			return nil, fmt.Errorf("failed to load from file: %w", err)
		}
	case "url":
		u, err := url.Parse(openAPISpecLocation)
		if err != nil {
			return nil, fmt.Errorf("invalid URL format: %w", err)
		}
		doc, err = loader.LoadFromURI(u)
		if err != nil {
			return nil, fmt.Errorf("failed to load from URL: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown source type: %s", sourceType)
	}

	// Conditional validation based on strictValidation flag
	if strictValidation {
		log.Println("Performing strict OpenAPI validation...")
		if err := doc.Validate(context.Background()); err != nil {
			return nil, fmt.Errorf("invalid OpenAPI specification (strict mode): %w", err)
		}
	} else {
		log.Println("Skipping strict validation (permissive mode)")
	}

	log.Printf("Loaded OpenAPI spec: %s v%s", doc.Info.Title, doc.Info.Version)
	return doc, nil
}

// detectSourceType determines whether the spec location is a URL or file path
func (s *OpenAPISource) detectSourceType(openAPISpecLocation string) string {
	u, err := url.Parse(openAPISpecLocation)
	if err == nil && u.Scheme != "" && (u.Scheme == "http" || u.Scheme == "https") {
		return "url"
	}
	return "file"
}

// GetSampleTool returns a sample OpenAPI MCP tool for testing and validation.
func (s *OpenAPISource) GetSampleTool() core.MakeMCPTool {
	return &OpenAPIMcpTool{}
}

// UnmarshalConfig reconstructs a MakeMCPApp from JSON data for the load command
func (s *OpenAPISource) UnmarshalConfig(data []byte) (*core.MakeMCPApp, error) {
	return core.UnmarshalConfigWithTypedParams[*OpenAPIMcpTool, *OpenAPIParams](data)
}

// OpenAPIHandlerInput defines how a particular endpoint is to be called
type OpenAPIHandlerInput struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
	Cookies     map[string]string `json:"cookies"`
	BodyAppend  map[string]any    `json:"bodyAppend"`
	ContentType string            `json:"contentType,omitempty"`
}

// NewOpenAPIHandlerInput creates a new OpenAPIHandlerInput
func NewOpenAPIHandlerInput(method, path string) OpenAPIHandlerInput {
	return OpenAPIHandlerInput{
		Method:     method,
		Path:       path,
		Headers:    make(map[string]string),
		Cookies:    make(map[string]string),
		BodyAppend: make(map[string]any),
	}
}

// OpenAPIMcpTool represents an MCP tool generated from an OpenAPI operation.
type OpenAPIMcpTool struct {
	core.McpTool
	OpenAPIHandlerInput *OpenAPIHandlerInput `json:"oapiHandlerInput,omitempty"`
	handler             func(
		ctx context.Context,
		request mcp.CallToolRequest,
		// TODO: refactor to get rid of mcp-go dependency
	) (*mcp.CallToolResult, error) `json:"-"`
}

// GetName returns the name of the OpenAPI MCP tool.
func (o *OpenAPIMcpTool) GetName() string {
	return o.McpTool.Name
}

// ToMcpTool returns the core MCP tool representation.
func (o *OpenAPIMcpTool) ToMcpTool() core.McpTool {
	return o.McpTool
}

// ToJSON returns a JSON representation of the OpenAPI MCP tool.
func (o *OpenAPIMcpTool) ToJSON() string {
	// TODO
	return ""
}

// GetHandler returns the MCP tool handler function for processing requests.
func (o *OpenAPIMcpTool) GetHandler() func(
	ctx context.Context,
	request mcp.CallToolRequest,
	// TODO: refactor to get rid of mcp-go dependency
) (*mcp.CallToolResult, error) {
	return o.handler
}

// createToolFromOperation creates a MakeMCPTool from an OpenAPI operation
func (s *OpenAPISource) createToolFromOperation(method, path string, operation *openapi3.Operation) OpenAPIMcpTool {
	toolName := s.getToolName(method, path, operation)
	toolInputSchema := s.getToolInputSchema(operation)
	toolAnnotations := s.getToolAnnotations(method, path, operation)

	// Create tool description
	description := operation.Description
	if description == "" {
		description = operation.Summary
	}
	if description == "" {
		description = fmt.Sprintf("%s %s", strings.ToUpper(method), path)
	}

	// Add schema documentation for non-JSON request bodies
	bodySchemaDoc := extractRequestBodySchemaDoc(operation)
	if bodySchemaDoc != "" {
		description = description + "\n\n" + bodySchemaDoc
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
			BodyAppend:  make(map[string]interface{}),
			ContentType: determineContentType(operation),
		},
	}

	return tool
}

// getToolName generates a tool name from operation ID or method and path.
func (s *OpenAPISource) getToolName(method string, path string, operation *openapi3.Operation) string {
	toolName := operation.OperationID
	if toolName == "" {
		toolName = fmt.Sprintf("%s_%s", method, path)
	}

	// Clean up the tool name by removing invalid characters
	toolName = strings.ReplaceAll(toolName, "{", "")
	toolName = strings.ReplaceAll(toolName, "}", "")
	toolName = strings.ReplaceAll(toolName, "/", "_")
	toolName = strings.ReplaceAll(toolName, "-", "_")
	toolName = strings.ToLower(toolName)

	return toolName
}

// getToolInputSchema creates the input schema for a tool.
func (s *OpenAPISource) getToolInputSchema(
	operation *openapi3.Operation,
) core.McpToolInputSchema {
	// TODO: refactor to get rid of mcp dependency
	genericProps := make(map[string]interface{})
	var required []string

	// Extract path, query, header, and cookie parameters
	for _, in := range []ParameterLocation{ParameterLocationPath, ParameterLocationQuery, ParameterLocationHeader, ParameterLocationCookie} {
		props, reqs := extractParametersByIn(operation, in)
		for paramName, prop := range props {
			prefixedName := fmt.Sprintf("%s__%s", in, paramName)
			genericProps[prefixedName] = map[string]interface{}{
				"type":        prop.Type,
				"description": prop.Description,
			}
			for _, reqName := range reqs {
				if reqName == paramName {
					required = append(required, prefixedName)
					break
				}
			}
		}
	}

	// Extract request body properties
	bodyProps, bodyReqs := extractRequestBodyProperties(operation)
	for paramName, prop := range bodyProps {
		prefixedName := fmt.Sprintf("body__%s", paramName)
		genericProps[prefixedName] = map[string]interface{}{
			"type":        prop.Type,
			"description": prop.Description,
		}
		for _, reqName := range bodyReqs {
			if reqName == paramName {
				required = append(required, prefixedName)
				break
			}
		}
	}

	return core.McpToolInputSchema{ // TODO: refactor to get rid of mcp dependency
		Type:       "object",
		Properties: genericProps,
		Required:   required,
	}
}

// getToolAnnotations returns tool annotations based on HTTP method and operation.
func (s *OpenAPISource) getToolAnnotations(
	method string, path string, operation *openapi3.Operation,
) core.McpToolAnnotation {
	// TODO: refactor to get rid of mcp dependency
	annotation := core.McpToolAnnotation{
		Title:           s.getToolName(method, path, operation),
		ReadOnlyHint:    nil,
		DestructiveHint: nil,
		IdempotentHint:  nil,
		OpenWorldHint:   nil,
	}

	methodUpper := strings.ToUpper(method)

	// ReadOnlyHint: GET, HEAD, OPTIONS are considered read-only and idempotent
	if methodUpper == "GET" || methodUpper == "HEAD" || methodUpper == "OPTIONS" {
		annotation.ReadOnlyHint = boolPtr(true)
		annotation.IdempotentHint = boolPtr(true)
	}

	// DestructiveHint: DELETE is considered destructive
	if methodUpper == "DELETE" {
		annotation.DestructiveHint = boolPtr(true)
	}

	// IdempotentHint: PUT is idempotent, POST is not
	if methodUpper == "PUT" {
		annotation.IdempotentHint = boolPtr(true)
	} else if methodUpper == "POST" {
		annotation.IdempotentHint = boolPtr(false)
	}

	return annotation
}

// =============================================================================
// TOOL INPUT SCHEMA GENERATION
// =============================================================================

// getSchemaTypeString returns the first OpenAPI type if present, otherwise defaults to "string".
func getSchemaTypeString(schema *openapi3.Schema) string {
	if schema != nil && schema.Type != nil && len(*schema.Type) > 0 {
		return (*schema.Type)[0]
	}
	return "string"
}

// extractParametersByIn extracts parameters of a given 'in' type (e.g., "path", "query", "header", "cookie")
// from an OpenAPI operation and returns properties and required fields.
func extractParametersByIn(operation *openapi3.Operation, in ParameterLocation) (map[string]ToolInputProperty, []string) {
	properties := make(map[string]ToolInputProperty)
	var required []string
	for _, paramRef := range operation.Parameters {
		if paramRef.Value == nil {
			continue
		}
		param := paramRef.Value
		if param.In == string(in) {
			typeName := getSchemaTypeString(param.Schema.Value)
			properties[param.Name] = ToolInputProperty{
				Type:        typeName,
				Description: param.Description,
				Location:    in, // Add the location field to record OpenAPI "in" value
			}
			if param.Required {
				required = append(required, param.Name)
			}
		}
	}
	return properties, required
}

// generateSchemaDocumentation creates human-readable schema documentation for XML/other content types.
func generateSchemaDocumentation(schema *openapi3.Schema, contentType string) string {
	if schema == nil || len(schema.Properties) == 0 {
		return fmt.Sprintf("Provide %s content as a string.", contentType)
	}

	var doc strings.Builder
	doc.WriteString(fmt.Sprintf("Expected %s structure:\n", contentType))

	for propName, propSchemaRef := range schema.Properties {
		propSchema := propSchemaRef.Value
		if propSchema == nil {
			continue
		}

		required := ""
		if contains(schema.Required, propName) {
			required = " (required)"
		}

		doc.WriteString(fmt.Sprintf("- %s: %s%s", propName, getSchemaTypeString(propSchema), required))
		if propSchema.Description != "" {
			doc.WriteString(fmt.Sprintf(" - %s", propSchema.Description))
		}
		doc.WriteString("\n")
	}

	doc.WriteString(fmt.Sprintf("\nProvide the complete %s as a string in the 'body' parameter.", contentType))
	return doc.String()
}

// contains checks if a string slice contains a specific string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// determineContentType returns the preferred content type for an operation's request body.
func determineContentType(operation *openapi3.Operation) string {
	if operation.RequestBody == nil || operation.RequestBody.Value == nil {
		return ""
	}

	// Priority order for content types
	contentTypes := []string{"application/json", "*/*", "text/xml", "application/xml", "text/plain"}

	for _, contentType := range contentTypes {
		if _, exists := operation.RequestBody.Value.Content[contentType]; exists {
			return contentType
		}
	}

	// Return first available content type if no priority match
	for contentType := range operation.RequestBody.Value.Content {
		return contentType
	}

	return ""
}

// extractRequestBodySchemaDoc extracts schema documentation for non-JSON content types.
func extractRequestBodySchemaDoc(operation *openapi3.Operation) string {
	if operation.RequestBody == nil || operation.RequestBody.Value == nil {
		return ""
	}

	// Check for non-JSON content types that need schema documentation
	nonJSONContentTypes := []string{"text/xml", "application/xml", "text/plain"}

	for _, contentType := range nonJSONContentTypes {
		if media, exists := operation.RequestBody.Value.Content[contentType]; exists {
			if media.Schema != nil && media.Schema.Value != nil {
				return generateSchemaDocumentation(media.Schema.Value, contentType)
			}
		}
	}

	return ""
}

// extractRequestBodyProperties extracts properties and required fields from the request body schema (if present).
func extractRequestBodyProperties(operation *openapi3.Operation) (map[string]ToolInputProperty, []string) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	if operation.RequestBody == nil || operation.RequestBody.Value == nil {
		return properties, required
	}

	// Check for supported content types in priority order
	contentTypes := []string{"application/json", "*/*", "text/xml", "application/xml"}

	for _, contentType := range contentTypes {
		if media, exists := operation.RequestBody.Value.Content[contentType]; exists {
			return extractPropertiesFromMedia(media, contentType)
		}
	}

	// If no recognized content type found, try the first available one
	for contentType, media := range operation.RequestBody.Value.Content {
		return extractPropertiesFromMedia(media, contentType)
	}

	return properties, required
}

// extractPropertiesFromMedia extracts properties from a media type
func extractPropertiesFromMedia(media *openapi3.MediaType, contentType string) (map[string]ToolInputProperty, []string) {
	if contentType == "application/json" || contentType == "*/*" {
		return extractJSONProperties(media)
	}
	return extractNonJSONProperties(media, contentType)
}

// extractJSONProperties handles JSON/generic content types
func extractJSONProperties(media *openapi3.MediaType) (map[string]ToolInputProperty, []string) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	if media.Schema != nil && media.Schema.Value != nil {
		schema := media.Schema.Value
		for propName, propSchemaRef := range schema.Properties {
			propSchema := propSchemaRef.Value
			if propSchema == nil {
				continue
			}
			properties[propName] = ToolInputProperty{
				Type:        getSchemaTypeString(propSchema),
				Description: propSchema.Description,
				Location:    "body",
			}
		}
		if schema.Required != nil {
			required = append(required, schema.Required...)
		}
	}
	return properties, required
}

// extractNonJSONProperties handles XML and other structured types
func extractNonJSONProperties(media *openapi3.MediaType, contentType string) (map[string]ToolInputProperty, []string) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	if media.Schema != nil && media.Schema.Value != nil && len(media.Schema.Value.Properties) > 0 {
		// Has schema properties - treat like JSON for unknown content types
		return extractJSONProperties(media)
	}

	// No schema or properties - single body parameter
	properties["body"] = ToolInputProperty{
		Type:        "string",
		Description: fmt.Sprintf("%s request body", contentType),
		Location:    "body",
	}
	required = append(required, "body")

	return properties, required
}

// AttachToolHandlers adds tool handler functions to app.
func (s *OpenAPISource) AttachToolHandlers(app *core.MakeMCPApp) error {
	// Type assert to get OpenAPI-specific parameters
	openAPIParams, ok := app.SourceParams.(*OpenAPIParams)
	if !ok {
		return fmt.Errorf("expected OpenAPIParams, got %T", app.SourceParams)
	}

	apiClient := NewAPIClient(openAPIParams.BaseURL, openAPIParams.Timeout)
	for i, tool := range app.Tools {
		// TODO: ugly type assertion
		openApiTool := tool.(*OpenAPIMcpTool)
		openApiTool.handler = GetHandlerFunction(openApiTool, apiClient)
		app.Tools[i] = openApiTool
	}
	return nil
}

// buildRequestURL constructs the full URL with path and query parameters..
func buildRequestURL(baseURL, path string, params ToolParams) string {
	pathWithParams := substitutePathParams(path, params.Path)
	fullURL := baseURL + pathWithParams

	// Add query parameters for all HTTP methods
	if len(params.Query) > 0 {
		encodedQuery := encodeQueryParams(params.Query)
		if encodedQuery != "" {
			fullURL = fullURL + "?" + encodedQuery
		}
	}

	return fullURL
}

// buildRequestBody prepares the request body for non-GET/DELETE methods based on content type.
func buildRequestBody(params ToolParams, method, contentType string) (io.Reader, error) {
	if len(params.Body) > 0 && !(method == http.MethodGet || method == http.MethodDelete) {
		switch contentType {
		case "text/xml", "application/xml":
			// For XML, expect a single "body" parameter with XML string
			if bodyContent, exists := params.Body["body"]; exists {
				if bodyStr, ok := bodyContent.(string); ok {
					return strings.NewReader(bodyStr), nil
				}
				return nil, fmt.Errorf("XML body parameter must be a string containing valid XML")
			}
			return nil, fmt.Errorf("XML content type requires a 'body' parameter with XML string")
		case "text/plain":
			// For plain text, expect a single "body" parameter
			if bodyContent, exists := params.Body["body"]; exists {
				if bodyStr, ok := bodyContent.(string); ok {
					return strings.NewReader(bodyStr), nil
				}
				return nil, fmt.Errorf("plain text body parameter must be a string")
			}
			return nil, fmt.Errorf("plain text content type requires a 'body' parameter with text string")
		default:
			// Default to JSON serialization for application/json, */*, and others
			jsonBody, err := json.Marshal(params.Body)
			if err != nil {
				return nil, err
			}
			return bytes.NewReader(jsonBody), nil
		}
	}
	return nil, nil
}

// setRequestHeaders applies headers and cookies to the HTTP request with appropriate content type.
func setRequestHeaders(req *http.Request, params ToolParams, hasBody bool, contentType string) {
	if hasBody && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else if hasBody {
		req.Header.Set("Content-Type", "application/json") // Default fallback
	}

	// Set headers
	for k, v := range params.Header {
		req.Header.Set(k, fmt.Sprintf("%v", v))
	}

	// Set cookies
	if len(params.Cookie) > 0 {
		var cookieStrings []string
		for k, v := range params.Cookie {
			cookieStrings = append(cookieStrings, fmt.Sprintf("%s=%v", k, v))
		}
		req.Header.Set("Cookie", strings.Join(cookieStrings, "; "))
	}
}

// GetHandlerFunction creates an MCP tool handler function for an OpenAPI operation.
//
// This function returns a handler that processes MCP tool requests and converts them
// into HTTP requests to the underlying API. It handles parameter parsing, HTTP request
// construction, execution, and response formatting.
//
// Parameters:
//   - makeMcpTool: MCP tool configuration containing OpenAPI operation details
//   - apiClient: HTTP client configured with base URL and other settings
//
// Returns:
//   - A function that handles MCP tool requests with the signature:
//     func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
//
// Request Processing Flow:
//  1. Parse prefixed parameters from the MCP request (e.g., path__user_id)
//  2. Group parameters by location (path, query, header, cookie, body)
//  3. Substitute path parameters in the URL template
//  4. Encode query parameters for GET/DELETE requests
//  5. Marshal body parameters to JSON for POST/PUT requests
//  6. Set headers and cookies on the HTTP request
//  7. Execute the HTTP request
//  8. Format and return the response
//
// Parameter Format:
// The handler expects parameters in prefix format:
//   - path__id: substituted into URL path placeholders
//   - query__limit: added as URL query parameters
//   - header__authorization: set as HTTP headers
//   - cookie__session_id: set as HTTP cookies
//   - body__email: included in JSON request body
//
// Response Format:
// Returns a formatted text result containing:
//   - HTTP method and final URL
//   - Response status code
//   - Response body content
func GetHandlerFunction(makeMcpTool *OpenAPIMcpTool, apiClient *APIClient) func(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters using prefix approach
		argsRaw := request.GetArguments()
		params := parsePrefixedParameters(argsRaw)

		method := makeMcpTool.OpenAPIHandlerInput.Method

		// Build URL and body using helper functions
		fullURL := buildRequestURL(apiClient.BaseURL, makeMcpTool.OpenAPIHandlerInput.Path, params)
		bodyReader, err := buildRequestBody(params, method, makeMcpTool.OpenAPIHandlerInput.ContentType)
		if err != nil {
			return nil, err
		}

		// Create the HTTP request
		req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
		if err != nil {
			return nil, err
		}

		// Apply headers and cookies using helper function
		setRequestHeaders(req, params, bodyReader != nil, makeMcpTool.OpenAPIHandlerInput.ContentType)

		// Execute request
		resp, err := apiClient.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("Failed to close response body: %v", err)
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		// Format the response for the MCP client
		result := fmt.Sprintf(
			"HTTP %s %s\nStatus: %d\nResponse: %s",
			method, fullURL, resp.StatusCode, string(body),
		)

		// TODO: what about other response types?
		return mcp.NewToolResultText(result), nil
	}
}
