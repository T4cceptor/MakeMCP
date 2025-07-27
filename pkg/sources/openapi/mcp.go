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
	"strings"
	"time"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
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
func (s *OpenAPISource) ParseParams(input *core.CLIParamsInput) (core.AppParams, error) {
	return ParseFromCLIInput(input)
}

// UnmarshalConfig reconstructs a MakeMCPApp from JSON data for the load command
func (s *OpenAPISource) UnmarshalConfig(data []byte) (*core.MakeMCPApp, error) {
	return core.UnmarshalConfigWithTypedParams[*OpenAPIMcpTool, *OpenAPIParams](data)
}

// Parse converts an OpenAPI specification into a MakeMCPApp configuration
func (s *OpenAPISource) Parse(params core.AppParams) (*core.MakeMCPApp, error) {
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
	openAPIParams, ok := app.AppParams.(*OpenAPIParams)
	if !ok {
		return fmt.Errorf("expected OpenAPIParams, got %T", app.AppParams)
	}

	apiClient := NewAPIClient(openAPIParams.BaseURL, openAPIParams.Timeout)
	for i, tool := range app.Tools {
		// TODO: ugly type assertion
		openApiTool := tool.(*OpenAPIMcpTool)
		openApiTool.handler = GetOpenAPIHandler(openApiTool, apiClient)
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
func buildRequestBody(params ToolParams, tool *OpenAPIMcpTool) (io.Reader, error) {
	if len(params.Body) > 0 {
		// Use the global content type registry to handle body building
		registry := NewContentTypeRegistry()
		handler := registry.GetHandler(tool.OpenAPIHandlerInput.ContentType)
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

// GetOpenAPIHandler creates a transport-agnostic MCP tool handler function for an OpenAPI operation.
//
// This function returns a handler that processes abstract tool execution contexts and converts them
// into HTTP requests to the underlying API. It handles parameter parsing, HTTP request
// construction, execution, and response formatting with rich metadata.
//
// Parameters:
//   - makeMcpTool: MCP tool configuration containing OpenAPI operation details
//   - apiClient: HTTP client configured with base URL and other settings
//
// Returns:
//   - A transport-agnostic handler with the signature:
//     func(ctx context.Context, request ToolExecutionContext) (ToolExecutionResult, error)
//
// Request Processing Flow:
//  1. Parse prefixed parameters from the execution context (e.g., path__user_id)
//  2. Group parameters by location (path, query, header, cookie, body)
//  3. Substitute path parameters in the URL template
//  4. Encode query parameters for GET/DELETE requests
//  5. Marshal body parameters using appropriate content type handlers
//  6. Set headers and cookies on the HTTP request
//  7. Execute the HTTP request with timing
//  8. Create rich result with metadata for processors
//
// The result includes comprehensive metadata that processors can use for:
//   - Content formatting (JSON vs text)
//   - Error handling and display
//   - Performance monitoring
//   - Security redaction
//   - Response caching decisions
func GetOpenAPIHandler(makeMcpTool *OpenAPIMcpTool, apiClient *APIClient) core.MakeMcpToolHandler {
	return func(ctx context.Context, request core.ToolExecutionContext) (core.ToolExecutionResult, error) {
		startTime := time.Now()

		// Parse parameters using prefix approach
		params := parsePrefixedParameters(request.GetParameters())
		method := makeMcpTool.OpenAPIHandlerInput.Method

		// Build URL and body using helper functions
		fullURL := buildRequestURL(apiClient.BaseURL, makeMcpTool.OpenAPIHandlerInput.Path, params)
		bodyReader, err := buildRequestBody(params, makeMcpTool)
		if err != nil {
			return core.NewBasicExecutionResult("Error: ", fmt.Errorf("failed to build request body: %w", err)), nil
		}

		// Create the HTTP request
		req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
		if err != nil {
			return core.NewBasicExecutionResult("Error: ", fmt.Errorf("failed to create HTTP request: %w", err)), nil
		}

		// Apply headers and cookies using helper function
		setRequestHeaders(req, params, bodyReader != nil, makeMcpTool.OpenAPIHandlerInput.ContentType)

		// Execute request with timing
		requestStartTime := time.Now()
		resp, err := apiClient.HTTPClient.Do(req)
		responseTime := time.Since(requestStartTime)

		if err != nil {
			return core.NewBasicExecutionResult("Error: ", fmt.Errorf("HTTP request failed: %w", err)), nil
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("Failed to close response body: %v", err)
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return core.NewBasicExecutionResult(
				"Error: ",
				fmt.Errorf("failed to read response body: %w", err),
			), nil
		}

		// Format the response for the client (same format as old handler for compatibility)
		result := fmt.Sprintf(
			"HTTP %s %s\nStatus: %d\nResponse: %s",
			req.Method, req.URL, resp.StatusCode, string(body),
		)

		// Create execution result with rich metadata
		executionResult := core.NewBasicExecutionResult(result, nil)
		metadata := executionResult.GetMetadata()

		// Set rich metadata for processors using Metadata interface
		metadata.Set("executionTime", time.Since(startTime))
		metadata.Set("httpStatus", resp.StatusCode)
		metadata.Set("responseTime", responseTime)
		metadata.Set("httpMethod", method)
		metadata.Set("finalURL", req.URL.String())
		metadata.Set("responseHeaders", resp.Header)

		// Set content type based on response
		contentType := resp.Header.Get("Content-Type")
		if contentType != "" {
			metadata.Set("actualContentType", contentType)
		}

		// Set processing hints as custom metadata for processors to use
		if strings.Contains(contentType, "application/json") {
			metadata.Set("isJsonData", true)
			metadata.Set("preferredFormat", "json")
		}

		// Mark errors based on HTTP status
		if resp.StatusCode >= 400 {
			metadata.Set("isErrorResponse", true)
			metadata.Set("shouldRedact", true) // Error responses might contain sensitive info
		}

		// Large responses should be truncated for display
		if len(body) > 10000 {
			metadata.Set("shouldTruncate", true)
			metadata.Set("maxDisplaySize", 10000)
		}

		return executionResult, nil
	}
}
