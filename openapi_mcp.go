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
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Uses kin-openapi/openapi3 library to load OpenAPI specs from URL or file
func loadOpenAPISpec(openAPISpecLocation string) *openapi3.T {
	log.Println("Loading OpenAPI spec from:", openAPISpecLocation)
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

// Creates a slice of MCPToolConfig from OpenAPI 3 specification
// Uses a provided apiClient to create a handler function for the tool
func GetToolConfigsFromOpenAPI(openapiSpec *openapi3.T, apiClient *APIClient) []MCPToolConfig {
	var toolConfigs []MCPToolConfig
	for path, pathItem := range openapiSpec.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			toolName := operation.OperationID
			if toolName == "" {
				toolName = fmt.Sprintf("%s_%s", method, path)
			}
			description := operation.Summary
			if operation.Description != "" {
				desc := strings.ReplaceAll(operation.Description, "\n", "\n\n")
				description = fmt.Sprintf("%v\n\n\n%v", description, desc)
			}
			log.Println("Tool description:", description)

			// Use helper methods for argument extraction and convert to ToolOptions
			options := []mcp.ToolOption{
				mcp.ToolOption(
					mcp.WithDescription(description),
				),
			}
			options = append(options, extractParameters(operation)...)
			options = append(options, extractRequestBodyArguments(operation)...)
			handler := makeOpenAPIToolHandlerWithClient(
				method,
				path,
				operation,
				apiClient,
			)
			toolConfigs = append(toolConfigs, MCPToolConfig{
				Name:        toolName,
				Description: description,
				Options:     options,
				Handler:     handler,
			})
		}
	}
	return toolConfigs
}

// TODO:
// func GetMakeMCPNamespace(openapiSpec *openapi3.T) MakeMCPNamespace {
// 	return nil
// }

func GetMCPServerFromToolConfigs(
	servername string, version string, toolConfigs []MCPToolConfig,
) *server.MCPServer {
	mcp_server := server.NewMCPServer(
		servername,
		version,
		server.WithToolCapabilities(true),
	)
	for _, cfg := range toolConfigs {
		tool := mcp.NewTool(
			cfg.Name,
			cfg.Options...,
		)
		mcp_server.AddTool(tool, cfg.Handler)
		log.Printf("Registered TOOL: %s", tool.Name)
	}
	return mcp_server
}

// Creates a new MCP server using the provided MakeConfig
// Accepts baseURL as a parameter and initializes an HTTP client for API calls
func HandleOpenAPI(
	sourceURI string,
	baseURL string,
	transport TransportType,
	configOnly bool,
	port string,
) {
	var openapiSpec *openapi3.T = loadOpenAPISpec(sourceURI)
	log.Printf("\nOpenAPI doc loaded: %#v\n\n", openapiSpec.Info.Title)

	// var makemcpNamespace MakeMCPNamespace = GetMakeMCPNamespace(openapiSpec)
	// TODO: store makemcpNamespace as file (JSON ? or yaml)

	// if "config-only" flag was provided we exit here
	if configOnly {
		log.Printf("Created config file at:\n") // TODO: provide path to config file
		log.Println("Exiting...")
		os.Exit(0)
	}

	// Note: below code should be highly re-usable
	// TODO: read makemcpNamespace from file -> we can skip that here actually
	// TODO: get ToolConfigs including handlerFunctions from MakeMCPNamespace
	// TODO: use below code to create server (this one should be fairly easy)

	// Step 1: Collect tool configs
	apiClient := NewAPIClient(baseURL)
	toolConfigs := GetToolConfigsFromOpenAPI(openapiSpec, apiClient)
	mcp_server := GetMCPServerFromToolConfigs(
		openapiSpec.Info.Title,
		openapiSpec.Info.Version,
		toolConfigs,
	)

	// Start the MCP server
	switch transport {
	case TransportTypeHTTP:
		log.Println("Starting as http MCP server...")
		streamable_server := server.NewStreamableHTTPServer(mcp_server)
		streamable_server.Start(fmt.Sprintf(":%s", port)) // TODO: make port configurable
	case TransportTypeStdio:
		log.Println("Starting as stdio MCP server...")
		if err := server.ServeStdio(mcp_server); err != nil {
			log.Printf("Server error: %v\n", err)
		}
	default:
		// TODO: raise error ?!
	}
}

// NewAPIClient creates a new APIClient with the given baseURL.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

// Displays the provided tool request
func ShowRequest(request mcp.CallToolRequest) {
	// Print the current request for debugging
	log.Printf("Current request: %+v\n", request)

	// TODO: display all relevant information here
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

		// Debugging
		ShowRequest(request) // for debugging
		log.Println("fullURL: ", fullURL)
		log.Println("args: ", args)

		// 1. check for URL parsing -> based on path params
		// 2. check for headers, method, body
		// 3. is there anything else?
		//		- cookies - basically just headers, but could actually deserve their own config type
		//		- response ? - should this be known in advance?
		//		-
		// understand "operation" better!
		// based on this we should create a structure

		// TODO: double check this
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
		// TODO: why only on Get and Delete?
		if len(params) > 0 && (method == http.MethodGet || method == http.MethodDelete) {
			fullURL = fullURL + "?" + params.Encode()
		}

		// Create the HTTP request
		// TODO: handle auth -> should be forwarded?
		// -> check how auth is handled within MCP
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

		// TODO: check this this is the most appropriate way to return the request result
		result := fmt.Sprintf("HTTP %s %s\nStatus: %d\nResponse: %s", method, fullURL, resp.StatusCode, string(body))
		return mcp.NewToolResultText(result), nil
	}
}

// Extracts OpenAPI parameters into mcp.ToolOption slice
func extractParameters(operation *openapi3.Operation) []mcp.ToolOption {
	var options []mcp.ToolOption
	for _, paramRef := range operation.Parameters {
		if paramRef.Value == nil {
			// TODO: is this even possible? What would that mean?
			continue
		}
		param := paramRef.Value
		paramType := param.Schema.Value.Type
		log.Println("paramType: ", paramType)
		// Only WithString for now, but could be extended for types
		options = append(
			options,
			mcp.WithString(param.Name),
		)
	}
	return options
}

// Extracts OpenAPI request body fields into mcp.ToolOption slice
func extractRequestBodyArguments(operation *openapi3.Operation) []mcp.ToolOption {
	var options []mcp.ToolOption
	if operation.RequestBody != nil && operation.RequestBody.Value != nil {
		for contentType, media := range operation.RequestBody.Value.Content {
			if contentType == "application/json" && media.Schema != nil && media.Schema.Value != nil {
				for propName := range media.Schema.Value.Properties {
					options = append(
						options,
						mcp.WithString(propName),
					)
				}
			}
		}
	}
	return options
}

//  1. Translate OpenAPI spec into our internal config
//     -> we should store the original definition of the operation as well
func GetMCPServerConfigFromOpenAPISpecs() {

}
