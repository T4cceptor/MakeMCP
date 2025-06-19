package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Uses kin-openapi/openapi3 library to load OpenAPI specs from URL or file
func loadOpenAPISpec(openAPISpecLocation string) *openapi3.T {
	loader := openapi3.NewLoader()
	sourceType := detectSourceType(openAPISpecLocation)
	switch sourceType {
	case "file":
		doc, err := loader.LoadFromFile(openAPISpecLocation)
		if err != nil {
			log.Fatal(err)
		}
		return doc
	case "url":
		u, err := url.Parse(openAPISpecLocation)
		if err != nil {
			log.Fatal(err)
		}
		doc, err := loader.LoadFromURI(u)
		if err != nil {
			log.Fatal(err)
		}
		return doc
	default:
		log.Fatalf("Unknown sourceType: %s (must be 'file' or 'url')", sourceType)
		return nil
	}
}

// Determines the sourceType based on the provided openAPISpecLocation
func detectSourceType(openAPISpecLocation string) string {
	u, err := url.Parse(openAPISpecLocation)
	if err == nil && u.Scheme != "" && (u.Scheme == "http" || u.Scheme == "https") {
		return "url"
	}
	return "file"
}

// Creates a new MCP server using the provided MakeConfig
// Accepts baseURL as a parameter and initializes an HTTP client for API calls
func McpFromOpenAPISpec(sourceURI string, baseURL string) {
	log.Printf("Starting MCP server...")

	openapiSpec := loadOpenAPISpec(sourceURI)
	log.Printf("\nOpenAPI doc loaded: %#v\n\n", openapiSpec.Info.Title)

	apiClient := NewAPIClient(baseURL)

	s := server.NewMCPServer(
		openapiSpec.Info.Title,
		openapiSpec.Info.Version,
		server.WithToolCapabilities(true),
	)

	// Iterate over OpenAPI paths and methods, register each as a tool
	for path, pathItem := range openapiSpec.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			toolName := operation.OperationID
			if toolName == "" {
				toolName = fmt.Sprintf("%s_%s", method, path)
			}
			// description := fmt.Sprintf(
			// 	"Summary: %s\nDescription: %s\n",
			// 	operation.Summary,
			// 	operation.Description,
			// )
			tool := mcp.NewTool(
				toolName,
				mcp.WithDescription(operation.Summary),
				// TODO: add argument definitions here based on operation.Parameters
			)
			s.AddTool(
				tool,
				makeOpenAPIToolHandlerWithClient(
					method,
					path,
					operation,
					apiClient,
				),
			)
			log.Printf("Registered TOOL: %s (%s %s)", tool.Name, method, path)
		}
	}

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		log.Printf("Server error: %v\n", err)
	}
}

// APIClient struct to encapsulate baseURL and http.Client
type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewAPIClient creates a new APIClient with the given baseURL.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

// makeOpenAPIToolHandlerWithClient returns a handler function for an MCP tool that uses the provided APIClient
// to a specific HTTP method, path, and OpenAPI operation. The returned handler processes
// tool requests and can be extended to map request arguments to HTTP requests for the
// OpenAPI route.
//
// Parameters:
//   - method: The HTTP method (e.g., "GET", "POST").
//   - path: The OpenAPI path (e.g., "/users/{id}").
//   - operation: The OpenAPI operation object.
//
// Returns:
//   - A function that handles MCP tool requests for the given operation.
func makeOpenAPIToolHandlerWithClient(method, path string, operation *openapi3.Operation, apiClient *APIClient) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fullURL := apiClient.BaseURL + path

		// Prepare query parameters and body
		var bodyReader io.Reader
		params := url.Values{}
		args := request.GetArguments()
		if args != nil {
			for k, v := range args {
				if method == http.MethodGet || method == http.MethodDelete {
					params.Add(k, fmt.Sprintf("%v", v))
				} else {
					if bodyReader == nil {
						jsonBody, err := json.Marshal(args)
						if err != nil {
							return nil, err
						}
						bodyReader = io.NopCloser(bytes.NewReader(jsonBody))
					}
				}
			}
		}
		if len(params) > 0 && (method == http.MethodGet || method == http.MethodDelete) {
			fullURL = fullURL + "?" + params.Encode()
		}

		// Create the HTTP request
		req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
		if err != nil {
			return nil, err
		}
		if bodyReader != nil {
			req.Header.Set("Content-Type", "application/json")
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

		result := fmt.Sprintf("HTTP %s %s\nStatus: %d\nResponse: %s", method, fullURL, resp.StatusCode, string(body))
		return mcp.NewToolResultText(result), nil
	}
}
