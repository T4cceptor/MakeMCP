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
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// =============================================================================
// CONSTANTS AND UTILITY FUNCTIONS
// =============================================================================

// locationPrefixes defines the valid location prefixes for parameter names
var locationPrefixes = []string{"path__", "query__", "header__", "cookie__", "body__"}

// parseLocationPrefix extracts location and parameter name from a prefixed parameter name
// Returns the location, parameter name, and whether a valid prefix was found
func parseLocationPrefix(prefixedName string) (location string, paramName string, found bool) {
	for _, prefix := range locationPrefixes {
		if strings.HasPrefix(prefixedName, prefix) {
			location = strings.TrimSuffix(prefix, "__")
			paramName = strings.TrimPrefix(prefixedName, prefix)
			found = true
			return
		}
	}
	return "", "", false
}

// =============================================================================
// OPENAPI SPEC LOADING AND DETECTION
// =============================================================================

// loadOpenAPISpec loads an OpenAPI specification from a URL or local file path
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

// detectSourceType determines whether the spec location is a URL or file path
func detectSourceType(openAPISpecLocation string) string {
	u, err := url.Parse(openAPISpecLocation)
	if err == nil && u.Scheme != "" && (u.Scheme == "http" || u.Scheme == "https") {
		return "url"
	}
	return "file"
}

// =============================================================================
// TOOL NAME AND PARAMETER PROCESSING
// =============================================================================

// GetToolName generates a tool name from operation ID or method and path
func GetToolName(method string, path string, operation *openapi3.Operation) string {
	toolName := operation.OperationID
	if toolName == "" {
		toolName = fmt.Sprintf("%s_%s", method, path)
	}
	return toolName
}

// ParameterInfo holds structured information about a parameter
type ParameterInfo struct {
	Name        string
	Type        string
	Location    string
	Description string
	Required    bool
}

// extractParameterInfo extracts parameter information from the tool input schema using prefix approach
func extractParameterInfo(toolInputSchema mcp.ToolInputSchema) []ParameterInfo {
	var params []ParameterInfo

	for prefixedName, prop := range toolInputSchema.Properties {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}

		// Parse the prefixed parameter name
		location, paramName, found := parseLocationPrefix(prefixedName)
		if !found {
			continue
		}

		param := ParameterInfo{
			Name:     paramName,
			Type:     "string",
			Location: location,
		}

		if t, ok := propMap["type"].(string); ok {
			param.Type = t
		}
		if d, ok := propMap["description"].(string); ok {
			param.Description = d
		}

		for _, req := range toolInputSchema.Required {
			if req == prefixedName {
				param.Required = true
				break
			}
		}

		params = append(params, param)
	}
	return params
}

// groupParametersByLocation groups parameters by their location (path, query, header, cookie, body)
func groupParametersByLocation(params []ParameterInfo) map[string][]ParameterInfo {
	grouped := make(map[string][]ParameterInfo)
	for _, param := range params {
		grouped[param.Location] = append(grouped[param.Location], param)
	}
	return grouped
}

// formatParametersByLocation formats parameters grouped by location for display
func formatParametersByLocation(grouped map[string][]ParameterInfo) string {
	var sections []string

	// Order of locations for consistent output
	locationOrder := []string{"path", "query", "header", "cookie", "body"}
	locationTitles := map[string]string{
		"path":   "Path Parameters:",
		"query":  "Query Parameters:",
		"header": "Header Parameters:",
		"cookie": "Cookie Parameters:",
		"body":   "Body Parameters:",
	}

	for _, location := range locationOrder {
		if params, exists := grouped[location]; exists && len(params) > 0 {
			sections = append(sections, locationTitles[location])
			sections = append(sections, "")

			for _, param := range params {
				requiredStr := ""
				if param.Required {
					requiredStr = " (Required)"
				}

				descStr := ""
				if param.Description != "" {
					descStr = fmt.Sprintf(": %s", param.Description)
				}

				sections = append(sections, fmt.Sprintf("- %s%s%s", param.Name, requiredStr, descStr))
			}
			sections = append(sections, "")
		}
	}

	return strings.Join(sections, "\n")
}

// =============================================================================
// TOOL DESCRIPTION GENERATION
// =============================================================================

// replaceArgsSection removes or replaces any existing "Args:" or "Arguments:" sections in the description
func replaceArgsSection(description string) string {
	// Common patterns for argument sections that should be replaced
	// Using simpler patterns without lookaheads since Go doesn't support them
	argsSectionPatterns := []string{
		`(?i)\n\s*Args?:\s*\n[^#]*?(\n\n|\z)`,
		`(?i)\n\s*Arguments?:\s*\n[^#]*?(\n\n|\z)`,
		`(?i)\n\s*Parameters?:\s*\n[^#]*?(\n\n|\z)`,
	}

	for _, pattern := range argsSectionPatterns {
		re := regexp.MustCompile(pattern)
		description = re.ReplaceAllString(description, "\n\n")
	}

	// Clean up any extra newlines
	description = strings.TrimSpace(description)
	return description
}

// generateExampleInput creates example input JSON for the tool using prefix approach
func generateExampleInput(params []ParameterInfo) string {
	sampleValues := map[string]any{
		"string":   "example string",
		"integer":  42,
		"number":   3.14,
		"boolean":  true,
		"float":    2.718,
		"datetime": "2025-07-14T12:34:56Z",
		"url":      "https://example.com/resource",
		"email":    "user@example.com",
		"object":   map[string]any{"field": "value"},
	}

	exampleParams := make(map[string]any)
	for _, param := range params {
		var sampleValue any
		switch param.Type {
		case "string":
			if strings.Contains(strings.ToLower(param.Name), "email") {
				sampleValue = sampleValues["email"]
			} else if strings.Contains(strings.ToLower(param.Name), "url") {
				sampleValue = sampleValues["url"]
			} else if strings.Contains(strings.ToLower(param.Name), "date") || strings.Contains(strings.ToLower(param.Name), "time") {
				sampleValue = sampleValues["datetime"]
			} else {
				sampleValue = sampleValues["string"]
			}
		case "integer":
			sampleValue = sampleValues["integer"]
		case "number":
			sampleValue = sampleValues["number"]
		case "float":
			sampleValue = sampleValues["float"]
		case "boolean":
			sampleValue = sampleValues["boolean"]
		case "object":
			sampleValue = sampleValues["object"]
		default:
			sampleValue = fmt.Sprintf("<%s_value>", param.Name)
		}

		// Use prefix approach: location__parameterName
		prefixedName := fmt.Sprintf("%s__%s", param.Location, param.Name)
		exampleParams[prefixedName] = sampleValue
	}

	exampleJSON, _ := json.MarshalIndent(exampleParams, "", "  ")
	return string(exampleJSON)
}

// GetToolDescription creates a structured description for an OpenAPI operation
func GetToolDescription(
	method string,
	path string,
	operation *openapi3.Operation,
	toolInputSchema mcp.ToolInputSchema,
) string {
	// Start with operation summary and description
	description := operation.Summary
	if operation.Description != "" {
		desc := strings.ReplaceAll(operation.Description, "\n", "\n\n")
		if description != "" {
			description = fmt.Sprintf("%s\n\n%s", description, desc)
		} else {
			description = desc
		}
	}

	// Replace any existing "Args:" or "Arguments:" sections with our structured parameter sections
	// This handles cases where OpenAPI descriptions might contain informal argument lists
	description = replaceArgsSection(description)

	// Extract and group parameters
	params := extractParameterInfo(toolInputSchema)
	if len(params) > 0 {
		grouped := groupParametersByLocation(params)
		paramSection := formatParametersByLocation(grouped)
		if paramSection != "" {
			description = fmt.Sprintf("%s\n\n%s", description, strings.TrimSpace(paramSection))
		}
	}

	// Add example input with clear formatting instructions
	if len(params) > 0 {
		exampleInput := generateExampleInput(params)
		description = fmt.Sprintf("%s\n\nExample input:\n%s", description, exampleInput)

		// Add explicit instruction for AI agents about the required format
		description = fmt.Sprintf("%s\n\nIMPORTANT: When calling this tool, you must provide parameters using the prefix format where parameter names include their location (e.g., 'path__user_id', 'body__email') as demonstrated in the example above.", description)
	}

	log.Println("Tool description: \n", description)
	return description
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
func extractParametersByIn(operation *openapi3.Operation, in string) (map[string]ToolInputProperty, []string) {
	properties := make(map[string]ToolInputProperty)
	var required []string
	for _, paramRef := range operation.Parameters {
		if paramRef.Value == nil {
			continue
		}
		param := paramRef.Value
		if param.In == in {
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

// GetToolInputSchema generates an MCP tool input schema from an OpenAPI operation.
//
// This function extracts parameter definitions from various sources in the OpenAPI operation
// and creates a unified input schema using the prefix approach for parameter naming.
//
// Parameters:
//   - method: HTTP method (GET, POST, PUT, DELETE, etc.)
//   - path: API endpoint path (e.g., "/users/{id}")
//   - operation: OpenAPI operation object containing parameter definitions
//
// Returns:
//   - mcp.ToolInputSchema: Schema object defining the expected input format
//
// Parameter Sources:
//   - Path parameters: from operation.parameters with "in": "path"
//   - Query parameters: from operation.parameters with "in": "query"
//   - Header parameters: from operation.parameters with "in": "header"
//   - Cookie parameters: from operation.parameters with "in": "cookie"
//   - Body parameters: from operation.requestBody schema properties
//
// Prefix Format:
// All parameters are prefixed with their location for clarity:
//   - path__user_id: path parameter named "user_id"
//   - query__limit: query parameter named "limit"
//   - body__email: body parameter named "email"
//
// This approach eliminates ambiguity and makes it easier for AI agents to
// understand where each parameter should be placed in the HTTP request.
func GetToolInputSchema(method string, path string, operation *openapi3.Operation) mcp.ToolInputSchema {
	// Build the properties and required fields for the input schema using prefix approach
	genericProps := make(map[string]any)
	var required []string

	// Extract path, query, header, and cookie parameters
	for _, in := range []string{"path", "query", "header", "cookie"} {
		props, reqs := extractParametersByIn(operation, in)
		for paramName, prop := range props {
			// Use prefix approach: location__parameterName
			prefixedName := fmt.Sprintf("%s__%s", in, paramName)
			genericProps[prefixedName] = map[string]interface{}{
				"type":        prop.Type,
				"description": prop.Description,
			}
			// Update required list with prefixed names
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
		// Use prefix approach: body__parameterName
		prefixedName := fmt.Sprintf("body__%s", paramName)
		genericProps[prefixedName] = map[string]interface{}{
			"type":        prop.Type,
			"description": prop.Description,
		}
		// Update required list with prefixed names
		for _, reqName := range bodyReqs {
			if reqName == paramName {
				required = append(required, prefixedName)
				break
			}
		}
	}

	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: genericProps,
		Required:   required,
	}
}

// =============================================================================
// TOOL ANNOTATIONS
// =============================================================================

// GetToolAnnotations returns a ToolAnnotation struct with hints set based on HTTP method and OpenAPI operation details.
// This helps the MCP server and clients understand the behavior and intent of the tool (endpoint).
func GetToolAnnotations(method string, path string, operation *openapi3.Operation) mcp.ToolAnnotation {
	// Default: all hints are nil (unset)
	annotation := mcp.ToolAnnotation{
		Title:           GetToolName(method, path, operation),
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

	// IdempotentHint: PUT and DELETE are idempotent
	if methodUpper == "PUT" || methodUpper == "DELETE" {
		annotation.IdempotentHint = boolPtr(true)
	}

	// OpenWorldHint: If the operation description mentions 'open world', set this hint
	if operation.Description != "" && strings.Contains(strings.ToLower(operation.Description), "open world") {
		annotation.OpenWorldHint = boolPtr(true)
	}

	return annotation
}

// =============================================================================
// MAIN OPENAPI PROCESSING
// =============================================================================

// FromOpenAPISpecs creates a MakeMCPApp from the provided OpenAPI specification
func FromOpenAPISpecs(params CLIParams) MakeMCPApp {
	var openApiSpec *openapi3.T = loadOpenAPISpec(params.Specs)
	log.Printf("\nOpenAPI doc loaded: %#v\n\n", openApiSpec.Info.Title)

	// Create config objects for MCP server and MakeMCP config
	var app MakeMCPApp = NewMakeMCPApp(
		openApiSpec.Info.Title,
		openApiSpec.Info.Version,
		string(params.Transport),
	)
	app.OpenAPIConfig = &OpenAPIConfig{BaseUrl: params.BaseURL}

	// Iterate through OpenAPI specs to define MCP tools and related configs
	for path, pathItem := range openApiSpec.Paths.Map() {
		//
		for method, operation := range pathItem.Operations() {
			toolName := GetToolName(method, path, operation)
			toolInputSchema := GetToolInputSchema(method, path, operation)
			toolAnnotations := GetToolAnnotations(method, path, operation)

			// TODO: include toolname, toolInputSchema and toolAnnotations in tool description
			toolDescripton := GetToolDescription(
				method,
				path,
				operation,
				toolInputSchema,
			)

			toolSourceData, err := operation.MarshalJSON()
			if err != nil {
				log.Fatal("Error while creating tool source data.", err)
			}

			// TODO: we need to check when its a tool and when a resource/resource template
			handlerInput := NewOpenAPIHandlerInput(method, path)
			app.Tools = append(
				app.Tools,
				MakeMCPTool{
					Tool: mcp.Tool{
						Name:        toolName,
						Description: toolDescripton,
						InputSchema: toolInputSchema,
						Annotations: toolAnnotations,
					},
					ToolSource: ToolSource{
						URI:  "", // TODO
						Data: toolSourceData,
					},
					OpenAPIHandlerInput: &handlerInput,
				},
			)
		}
	}

	return app
}

// HandleOpenAPI processes OpenAPI specifications and starts the MCP server
func HandleOpenAPI(params CLIParams) {
	var app MakeMCPApp = FromOpenAPISpecs(params)

	// We could also load a json file and create handler functions from it
	AddOpenAPIHandlerFunctions(&app, NewAPIClient(params.BaseURL))

	// Store MakeMCPApp as json config
	SaveMakeMCPAppToFile(app)

	// if "config-only" flag was provided we exit here
	if params.ConfigOnly {
		log.Printf("Created config file at:\n") // TODO: provide path to config file
		log.Println("Exiting.")
		os.Exit(0)
	}

	// Step 1: Collect tool configs
	mcp_server := GetMCPServer(app)

	// Start the MCP server
	switch params.Transport {
	case TransportTypeHTTP:
		log.Println("Starting as http MCP server...")
		streamable_server := server.NewStreamableHTTPServer(mcp_server)
		streamable_server.Start(fmt.Sprintf(":%s", params.Port)) // TODO: make port configurable
	case TransportTypeStdio:
		log.Println("Starting as stdio MCP server...")
		if err := server.ServeStdio(mcp_server); err != nil {
			log.Printf("Server error: %v\n", err)
		}
	default:
		// TODO: raise error ?!
	}
}

// =============================================================================
// HTTP REQUEST HANDLING AND UTILITIES
// =============================================================================

// substitutePathParams substitutes path parameters in URL template
func substitutePathParams(path string, pathParams map[string]any) string {
	for k, v := range pathParams {
		placeholder := fmt.Sprintf("{%s}", k)
		path = strings.ReplaceAll(path, placeholder, fmt.Sprintf("%v", v))
	}
	return path
}

// encodeQueryParams encodes query parameters for URL
func encodeQueryParams(queryParams map[string]any) string {
	values := url.Values{}
	for k, v := range queryParams {
		values.Set(k, fmt.Sprintf("%v", v))
	}
	return values.Encode()
}

// parsePrefixedParameters parses parameters using prefix approach and returns SplitParams
func parsePrefixedParameters(argsRaw map[string]any) SplitParams {
	params := NewSplitParams()

	// Parse prefixed parameters and organize by location
	for prefixedName, value := range argsRaw {
		location, paramName, found := parseLocationPrefix(prefixedName)
		if !found {
			continue
		}

		// Add to appropriate location map in SplitParams
		switch location {
		case "path":
			params.Path[paramName] = value
		case "query":
			params.Query[paramName] = value
		case "header":
			params.Header[paramName] = value
		case "cookie":
			params.Cookie[paramName] = value
		case "body":
			params.Body[paramName] = value
		}
	}

	return params
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
//   1. Parse prefixed parameters from the MCP request (e.g., path__user_id)
//   2. Group parameters by location (path, query, header, cookie, body)
//   3. Substitute path parameters in the URL template
//   4. Encode query parameters for GET/DELETE requests
//   5. Marshal body parameters to JSON for POST/PUT requests
//   6. Set headers and cookies on the HTTP request
//   7. Execute the HTTP request
//   8. Format and return the response
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
func GetHandlerFunction(makeMcpTool MakeMCPTool, apiClient *APIClient) func(
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
		log.Println("fullURL: ", fullURL)

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
			bodyReader = io.NopCloser(bytes.NewReader(jsonBody))
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
		return mcp.NewToolResultText(result), nil
	}
}

// AddOpenAPIHandlerFunctions creates and attaches handler functions for each OpenAPI tool
func AddOpenAPIHandlerFunctions(app *MakeMCPApp, apiClient *APIClient) {
	for i := range app.Tools {
		app.Tools[i].HandlerFunction = GetHandlerFunction(app.Tools[i], apiClient)
	}
}
