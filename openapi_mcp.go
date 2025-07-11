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

// TODO: we need to abstract this better
func GetToolName(method string, path string, operation *openapi3.Operation) string {
	toolName := operation.OperationID
	if toolName == "" {
		toolName = fmt.Sprintf("%s_%s", method, path)
	}
	return toolName
}

// TODO: this needs to be improved based on provided data in the operation
func GetToolDescription(method string, path string, operation *openapi3.Operation) string {
	description := operation.Summary
	if operation.Description != "" {
		desc := strings.ReplaceAll(operation.Description, "\n", "\n\n")
		description = fmt.Sprintf("%v\n%v", description, desc)
	}
	// TODO: we need to improve the tool description by quite a bit, as the AI is not aware how the different parameters are supposed to be provided.

	description = fmt.Sprintf("%v\n Parameters are to be provided in the following schema: {'parameter_name': <name of the parameter, e.g. user_id>, 'parameter_value': <value of the parameter, e.g. 2>, 'location': <where the parameter is to be included, can be one of: path, query, header, cookie, body - will determine in which part of the request the param is placed>}", description)
	log.Println("Tool description: \n", description)

	// TODO: we need to add all sorts of parameters here, as they are supposed to be provided!

	/*
			Stops a current multipart upload and removes any parts of an incomplete upload, which would otherwise incur storage costs.

		Path Parameters:

		Bucket (Required): The destination bucket for the upload.

		Key (Required): Key of the object for which the multipart upload was initiated.

		Query Parameters:

		uploadId (Required): Upload ID that identifies the multipart upload.
		Responses:

		204 (Success): Success
		Content-Type: text/xml

		Response Properties:

		Example:

		404: NoSuchUpload
		Content-Type: text/xml
	*/
	return description
}

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

func GetToolInputSchema(method string, path string, operation *openapi3.Operation) mcp.ToolInputSchema {
	/* NOTE: LEAVE THIS AS-IS, it serves documentation for now and will be removed later once implementation is finished
	To implement GetToolInputSchema, you need to extract all the fields from the OpenAPI operation that define what input the tool expects. The following fields from the OpenAPI operation impact the input schema for the MCP tool:

	Fields Impacting the Input Schema
	- Path Parameters
		Defined in the parameters array with "in": "path".
		Example: /users/{id} → parameter id.
	- Query Parameters
		Defined in the parameters array with "in": "query".
		Example: /users?active=true → parameter active.
	- Header Parameters
		Defined in the parameters array with "in": "header".
		Example: X-Request-ID header.
	- Cookie Parameters
		Defined in the parameters array with "in": "cookie".
	- Request Body
		Defined in the requestBody field.
		Most commonly, this is a JSON object with a schema describing its properties.

	- Required Flags
		Both parameters and request body properties can be marked as required.

	Summary Table
		Source	OpenAPI Field	Example/Notes
		Path params	parameters (in=path)	/users/{id}
		Query params	parameters (in=query)	/users?active=true
		Header params	parameters (in=header)	X-Request-ID
		Cookie params	parameters (in=cookie)	session_id
		Request body	requestBody	JSON, form, etc.
		Required flags	required	Both in parameters and body

	Would you like a code template for extracting these into an MCP input schema?
	*/
	// Build the properties and required fields for the input schema
	properties := make(map[string]ToolInputProperty)
	var required []string

	// Extract path, query, header, and cookie parameters
	for _, in := range []string{"path", "query", "header", "cookie"} {
		props, reqs := extractParametersByIn(operation, in)
		for k, v := range props {
			properties[k] = v
		}
		required = append(required, reqs...)
	}

	// Extract request body properties
	bodyProps, bodyReqs := extractRequestBodyProperties(operation)
	for k, v := range bodyProps {
		properties[k] = v
	}
	required = append(required, bodyReqs...)

	// Convert properties to map[string]any for MCP schema compatibility
	genericProps := make(map[string]any)
	for k, v := range properties {
		genericProps[k] = v
	}

	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: genericProps,
		Required:   required,
	}
}

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

// Displays the provided tool request
func ShowRequest(request mcp.CallToolRequest) {
	// Print the current request for debugging
	log.Printf("Current request: %+v\n", request)
}

// Creates a MakeMCPApp from the provided OpenAPI specification
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
			toolDescripton := GetToolDescription(method, path, operation)

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

// HandleOpenAPI now takes a CLIParams struct for all CLI arguments.
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

func GetHandlerFunction(makeMcpTool MakeMCPTool, apiClient *APIClient) func(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// TODO
		fullURL := apiClient.BaseURL + makeMcpTool.OpenAPIHandlerInput.Path
		log.Println("fullURL: ", fullURL)

		method := makeMcpTool.OpenAPIHandlerInput.Method

		// Prepare query parameters and body
		var bodyReader io.Reader
		params := url.Values{}
		args := request.GetArguments()

		// Debugging
		ShowRequest(request) // for debugging
		log.Println("args: ", args)

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
		result := fmt.Sprintf(
			"HTTP %s %s\nStatus: %d\nResponse: %s",
			method, fullURL, resp.StatusCode, string(body),
		)
		return mcp.NewToolResultText(result), nil
	}
}

// Takes an MakeMCPApp and creates + attaches tool handler functions for each tool
func AddOpenAPIHandlerFunctions(app *MakeMCPApp, apiClient *APIClient) {
	// TODO

	// TODO: create and add handler here!
	// if we wanted to created the handler functions here we
	// need to KNOW and HANDLE the type of tools we are
	// integrating with -> e.g. REST API, CLI-tools, etc.
	for i := range app.Tools {
		app.Tools[i].HandlerFunction = GetHandlerFunction(app.Tools[i], apiClient)
	}
}
