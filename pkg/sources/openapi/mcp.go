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

	"github.com/T4cceptor/MakeMCP/pkg/config"
)

// OpenAPISource implements the sources.Source interface for OpenAPI specifications
type OpenAPISource struct{}

// Name returns the name of this source type
func (s *OpenAPISource) Name() string {
	return "openapi"
}

// Parse converts an OpenAPI specification into a MakeMCPApp configuration
func (s *OpenAPISource) Parse(params *config.CLIParams) (*config.MakeMCPApp, error) {
	// Extract OpenAPI-specific parameters
	var openAPIParams OpenAPIParams
	if err := openAPIParams.FromCLIParams(params); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI parameters: %w", err)
	}

	// Security checks
	if !params.DevMode {
		WarnURLSecurity(openAPIParams.Specs, "OpenAPI spec", false)
		WarnURLSecurity(openAPIParams.BaseURL, "Base URL", false)
	}
	app := config.NewMakeMCPApp("", "", openAPIParams.CLIParams.Transport)
	app.SourceType = s.Name() // Set source type
	app.CliParams = openAPIParams.CLIParams
	// both are useful, but it raises dependency concerns

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
	var tools []config.MakeMCPTool
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			tool, err := s.createToolFromOperation(method, path, operation, params.ToJSON())
			if err != nil {
				log.Printf("Warning: failed to create tool for %s %s: %v", method, path, err)
				continue
			}
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

func (s *OpenAPISource) GetSampleTool() config.MakeMCPTool {
	return &OpenAPIMcpTool{}
}

// UnmarshalConfig reconstructs a MakeMCPApp from JSON data for the load command
func (s *OpenAPISource) UnmarshalConfig(data []byte) (*config.MakeMCPApp, error) {
	return config.UnmarshalConfigWithTools[*OpenAPIMcpTool](data)
}

// OpenAPIHandlerInput defines how a particular endpoint is to be called
type OpenAPIHandlerInput struct {
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Headers    map[string]string `json:"headers"`
	Cookies    map[string]string `json:"cookies"`
	BodyAppend map[string]any    `json:"bodyAppend"`
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

type OpenAPIMcpTool struct {
	config.McpTool
	OpenAPIHandlerInput *OpenAPIHandlerInput `json:"oapiHandlerInput,omitempty"`
	handler             func(
		ctx context.Context,
		request mcp.CallToolRequest,
		// TODO: refactor to get rid of mcp-go dependency
	) (*mcp.CallToolResult, error) `json:"-"`
}

func (o *OpenAPIMcpTool) GetName() string {
	return o.McpTool.Name
}

func (o *OpenAPIMcpTool) ToMcpTool() config.McpTool {
	return o.McpTool
}

func (o *OpenAPIMcpTool) ToJSON() string {
	// TODO
	return ""
}

func (o *OpenAPIMcpTool) GetHandler() func(
	ctx context.Context,
	request mcp.CallToolRequest,
	// TODO: refactor to get rid of mcp-go dependency
) (*mcp.CallToolResult, error) {
	return o.handler
}

// createToolFromOperation creates a MakeMCPTool from an OpenAPI operation
func (s *OpenAPISource) createToolFromOperation(method, path string, operation *openapi3.Operation, specSource string) (OpenAPIMcpTool, error) {
	toolName := s.getToolName(method, path, operation)
	toolInputSchema := s.getToolInputSchema(method, path, operation)
	toolAnnotations := s.getToolAnnotations(method, path, operation)

	// Create tool description
	description := operation.Description
	if description == "" {
		description = operation.Summary
	}
	if description == "" {
		description = fmt.Sprintf("%s %s", strings.ToUpper(method), path)
	}

	// Create the tool
	tool := OpenAPIMcpTool{
		McpTool: config.McpTool{
			Name:        toolName,
			Description: description,
			InputSchema: toolInputSchema,
			Annotations: toolAnnotations,
		},
		OpenAPIHandlerInput: &OpenAPIHandlerInput{
			Method:     method,
			Path:       path,
			Headers:    make(map[string]string),
			Cookies:    make(map[string]string),
			BodyAppend: make(map[string]interface{}),
		},
	}

	return tool, nil
}

// getToolName generates a tool name from operation ID or method and path
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

// getToolInputSchema creates the input schema for a tool
func (s *OpenAPISource) getToolInputSchema(
	method string, path string, operation *openapi3.Operation,
) config.McpToolInputSchema {
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

	return config.McpToolInputSchema{ // TODO: refactor to get rid of mcp dependency
		Type:       "object",
		Properties: genericProps,
		Required:   required,
	}
}

// getToolAnnotations returns tool annotations based on HTTP method and operation
func (s *OpenAPISource) getToolAnnotations(
	method string, path string, operation *openapi3.Operation,
) config.McpToolAnnotation {
	// TODO: refactor to get rid of mcp dependency
	annotation := config.McpToolAnnotation{
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

// extractRequestBodyProperties extracts properties and required fields from the request body schema (if present).
// Only supports application/json bodies with object schemas for now.
func extractRequestBodyProperties(operation *openapi3.Operation) (map[string]ToolInputProperty, []string) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	if operation.RequestBody == nil || operation.RequestBody.Value == nil {
		return properties, required
	}

	for contentType, media := range operation.RequestBody.Value.Content {
		if contentType != "application/json" || media.Schema == nil || media.Schema.Value == nil {
			continue
		}
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
		// Add required fields from the schema
		if schema.Required != nil {
			required = append(required, schema.Required...)
		}
	}
	return properties, required
}

// AttachToolHandlers adds tool handler functions to app
func (s *OpenAPISource) AttachToolHandlers(app *config.MakeMCPApp) error {
	baseUrl := app.CliParams.GetFlag(baseUrlCommand.Name).(string)
	// JSON technically only knows "number", which is parsed as float64
	// since timeout is supposed to be in seconds, we just cast it here
	timeout := int(app.CliParams.GetFlag("timeout").(float64))
	apiClient := NewAPIClient(baseUrl, timeout)
	for i, tool := range app.Tools {
		// TODO: ugly type assertion
		openApiTool := tool.(*OpenAPIMcpTool)
		openApiTool.handler = GetHandlerFunction(*openApiTool, apiClient)
		app.Tools[i] = openApiTool
	}
	return nil
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
func GetHandlerFunction(makeMcpTool OpenAPIMcpTool, apiClient *APIClient) func(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

		// Parse parameters using prefix approach
		argsRaw := request.GetArguments()
		params := parsePrefixedParameters(argsRaw)

		// Substitute path parameters
		pathWithParams := substitutePathParams(makeMcpTool.OpenAPIHandlerInput.Path, params.Path)
		fullURL := apiClient.BaseURL + pathWithParams
		method := makeMcpTool.OpenAPIHandlerInput.Method

		// Prepare query parameters and body
		var bodyReader io.Reader

		// Encode query parameters
		if len(params.Query) > 0 && (method == http.MethodGet || method == http.MethodDelete) {
			encodedQuery := encodeQueryParams(params.Query)
			if encodedQuery != "" {
				fullURL = fullURL + "?" + encodedQuery
			}
		}

		// Prepare body for non-GET/DELETE
		if len(params.Body) > 0 && !(method == http.MethodGet || method == http.MethodDelete) {
			jsonBody, err := json.Marshal(params.Body)
			if err != nil {
				return nil, err
			}
			bodyReader = bytes.NewReader(jsonBody)
		}

		// Create the HTTP request
		req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
		if err != nil {
			return nil, err
		}
		if bodyReader != nil {
			req.Header.Set("Content-Type", "application/json")
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

		resp, err := apiClient.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

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
