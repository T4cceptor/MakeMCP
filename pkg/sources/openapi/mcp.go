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
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
	"github.com/mark3labs/mcp-go/mcp"
)

// OpenAPISource implements the sources.Source interface for OpenAPI specifications
type OpenAPISource struct {
	adapter *LibopenAPIAdapter
}

// NewOpenAPISource creates a new OpenAPISource instance
func NewOpenAPISource() *OpenAPISource {
	return &OpenAPISource{
		adapter: NewLibopenAPIAdapter(),
	}
}

// Name returns the name of this source type
func (s *OpenAPISource) Name() string {
	return "openapi"
}

// ParseParams converts raw CLI input into typed OpenAPI parameters
func (s *OpenAPISource) ParseParams(input *core.CLIParamsInput) (core.SourceParams, error) {
	return ParseFromCLIInput(input)
}

// UnmarshalConfig reconstructs a MakeMCPApp from JSON data for the load command
func (s *OpenAPISource) UnmarshalConfig(data []byte) (*core.MakeMCPApp, error) {
	return core.UnmarshalConfigWithTypedParams[*OpenAPIMcpTool, *OpenAPIParams](data)
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

	// Initialize adapter if not present
	if s.adapter == nil {
		s.adapter = NewLibopenAPIAdapter()
	}

	// Load the OpenAPI specification
	doc, err := s.adapter.LoadOpenAPISpec(openAPIParams.Specs, openAPIParams.StrictValidate)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	// Create the app configuration
	title, version := s.adapter.GetDocumentInfo(doc)
	if app.Name == "" {
		app.Name = title
	}
	if app.Version == "" {
		app.Version = version
	}

	// Convert OpenAPI operations to MCP tools
	openAPITools, err := s.adapter.CreateToolsFromDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to process operations: %w", err)
	}
	app.Tools = convertToMakeMCPTools(openAPITools)
	return &app, nil
}

func convertToMakeMCPTools(tools []OpenAPIMcpTool) []core.MakeMCPTool {
	result := make([]core.MakeMCPTool, len(tools))
	for i := range tools {
		result[i] = &tools[i]
	}
	return result
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

// buildRequestURL constructs the full URL with path and query parameters.
func buildRequestURL(baseURL, path string, params ToolParams) string {
	var url strings.Builder
	url.WriteString(baseURL)
	url.WriteString(substitutePathParams(path, params.Path))

	// Add query parameters for all HTTP methods
	if len(params.Query) > 0 {
		encodedQuery := encodeQueryParams(params.Query)
		if encodedQuery != "" {
			url.WriteString("?")
			url.WriteString(encodedQuery)
		}
	}
	return url.String()
}

// buildRequestBody prepares the request body for non-GET/DELETE methods using content-type handlers.
func buildRequestBody(params ToolParams, method, contentType string) (io.Reader, error) {
	readOnlyMethods := []string{http.MethodGet, http.MethodDelete}
	if len(params.Body) > 0 && !slices.Contains(readOnlyMethods, method) {
		// Use the global content type registry to handle body building
		registry := NewContentTypeRegistry()
		handler := registry.GetHandler(contentType)
		return handler.BuildRequestBody(params.Body)
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
		var cookies strings.Builder
		first := true
		for k, v := range params.Cookie {
			if !first {
				cookies.WriteString("; ")
			}
			cookies.WriteString(fmt.Sprintf("%s=%v", k, v))
			first = false
		}
		req.Header.Set("Cookie", cookies.String())
	}
}

func buildRequest(
	ctx context.Context,
	request mcp.CallToolRequest,
	makeMcpTool *OpenAPIMcpTool,
	apiClient *APIClient,
) (*http.Request, error) {
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
	return req, nil
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
		// Get request
		req, err := buildRequest(ctx, request, makeMcpTool, apiClient)
		if err != nil {
			return nil, err
		}

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
			req.Method, req.URL, resp.StatusCode, string(body),
		)

		// TODO: what about other response types?
		return mcp.NewToolResultText(result), nil
	}
}
