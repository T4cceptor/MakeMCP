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
	"log"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/urfave/cli/v3"

	"github.com/T4cceptor/MakeMCP/pkg/config"
)

// OpenAPISource implements the sources.Source interface for OpenAPI specifications
type OpenAPISource struct{}

// Name returns the name of this source type
func (s *OpenAPISource) Name() string {
	return "openapi"
}

// Parse converts an OpenAPI specification into a MakeMCPApp configuration
func (s *OpenAPISource) Parse(input string, baseConfig config.MakeMCPApp) (*config.MakeMCPApp, error) {
	// Load the OpenAPI specification
	doc, err := s.loadOpenAPISpec(input)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	// Create the app configuration
	app := baseConfig
	if app.Name == "" {
		app.Name = doc.Info.Title
	}
	if app.Version == "" {
		app.Version = doc.Info.Version
	}

	// Set source configuration
	app.Source = config.SourceConfig{
		Type: "openapi",
		Config: map[string]interface{}{
			"spec": input,
		},
	}

	// Convert OpenAPI operations to MCP tools
	var tools []config.MakeMCPTool
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			tool, err := s.createToolFromOperation(method, path, operation, input)
			if err != nil {
				log.Printf("Warning: failed to create tool for %s %s: %v", method, path, err)
				continue
			}
			tools = append(tools, tool)
		}
	}

	app.Tools = tools
	return &app, nil
}

// Validate checks if the input is a valid OpenAPI specification
func (s *OpenAPISource) Validate(input string) error {
	_, err := s.loadOpenAPISpec(input)
	return err
}

// GetDefaultConfig returns the default configuration for OpenAPI sources
func (s *OpenAPISource) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"spec":    "",
		"baseUrl": "",
	}
}

// loadOpenAPISpec loads an OpenAPI specification from a URL or local file path
func (s *OpenAPISource) loadOpenAPISpec(openAPISpecLocation string) (*openapi3.T, error) {
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
	
	// Validate the OpenAPI specification
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI specification: %w", err)
	}
	
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

// createToolFromOperation creates a MakeMCPTool from an OpenAPI operation
func (s *OpenAPISource) createToolFromOperation(method, path string, operation *openapi3.Operation, specSource string) (config.MakeMCPTool, error) {
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
	tool := config.MakeMCPTool{
		Tool: mcp.Tool{
			Name:        toolName,
			Description: description,
			InputSchema: toolInputSchema,
			Annotations: toolAnnotations,
		},
		OpenAPIHandlerInput: &config.OpenAPIHandlerInput{
			Method:     method,
			Path:       path,
			Headers:    make(map[string]string),
			Cookies:    make(map[string]string),
			BodyAppend: make(map[string]interface{}),
		},
		ToolSource: config.ToolSource{
			URI:  specSource,
			Data: []byte(fmt.Sprintf("%s %s", method, path)), // Simple representation
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
func (s *OpenAPISource) getToolInputSchema(method string, path string, operation *openapi3.Operation) mcp.ToolInputSchema {
	genericProps := make(map[string]interface{})
	var required []string

	// Extract path, query, header, and cookie parameters
	for _, in := range []string{"path", "query", "header", "cookie"} {
		props, reqs := s.extractParametersByIn(operation, in)
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
	bodyProps, bodyReqs := s.extractRequestBodyProperties(operation)
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

	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: genericProps,
		Required:   required,
	}
}

// getToolAnnotations returns tool annotations based on HTTP method and operation
func (s *OpenAPISource) getToolAnnotations(method string, path string, operation *openapi3.Operation) mcp.ToolAnnotation {
	annotation := mcp.ToolAnnotation{
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

// Helper functions (simplified versions of the original implementation)
func (s *OpenAPISource) extractParametersByIn(operation *openapi3.Operation, in string) (map[string]config.ToolInputProperty, []string) {
	props := make(map[string]config.ToolInputProperty)
	var required []string
	
	// This is a simplified implementation - in a real implementation,
	// you'd need to properly parse the OpenAPI parameter schemas
	for _, param := range operation.Parameters {
		if param.Value != nil && param.Value.In == in {
			props[param.Value.Name] = config.ToolInputProperty{
				Type:        "string", // Simplified - should derive from schema
				Description: param.Value.Description,
				Location:    in,
			}
			if param.Value.Required {
				required = append(required, param.Value.Name)
			}
		}
	}
	
	return props, required
}

func (s *OpenAPISource) extractRequestBodyProperties(operation *openapi3.Operation) (map[string]config.ToolInputProperty, []string) {
	props := make(map[string]config.ToolInputProperty)
	var required []string
	
	// This is a simplified implementation - in a real implementation,
	// you'd need to properly parse the request body schema
	if operation.RequestBody != nil && operation.RequestBody.Value != nil {
		// Simplified: assume JSON content type
		if content, ok := operation.RequestBody.Value.Content["application/json"]; ok {
			if content.Schema != nil && content.Schema.Value != nil {
				for propName, propSchema := range content.Schema.Value.Properties {
					props[propName] = config.ToolInputProperty{
						Type:        "string", // Simplified - should derive from schema
						Description: propSchema.Value.Description,
						Location:    "body",
					}
				}
				required = content.Schema.Value.Required
			}
		}
	}
	
	return props, required
}

// GetCommand returns the CLI command for this source
func (s *OpenAPISource) GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "openapi",
		Usage: "Use OpenAPI specifications to launch an MCP server locally.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "specs",
				Aliases: []string{"s"},
				Value:   "",
				Usage:   "Where to find the OpenAPI specification - can be either a properly formed URL, including protocol, or a file path to a JSON file.",
			},
			&cli.StringFlag{
				Name:    "base-url",
				Aliases: []string{"b"},
				Value:   "",
				Usage:   "Base URL of the OpenAPI specified API. This will be called when invoking the tools.",
			},
			&cli.StringFlag{
				Name:    "transport",
				Aliases: []string{"t"},
				Value:   string(config.TransportTypeStdio),
				Usage:   "Used transport protocol for this MCP server - can be either stdio or http.",
			},
			&cli.BoolFlag{
				Name:    "config-only",
				Aliases: []string{"oc"},
				Value:   false,
				Usage:   "If set to true only creates a config file and exits, no server will be started.",
			},
			&cli.StringFlag{
				Name:  "port",
				Value: "8080",
				Usage: "Defines the port on which the HTTP server is started, ignored if transport is set to stdio.",
			},
			&cli.BoolFlag{
				Name:  "dev-mode",
				Value: false,
				Usage: "Enable development mode - suppresses security warnings for local/private URLs. Use only for local development.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			params := config.CLIParams{
				Specs:      cmd.String("specs"),
				BaseURL:    cmd.String("base-url"),
				Transport:  config.TransportType(cmd.String("transport")),
				ConfigOnly: cmd.Bool("config-only"),
				Port:       cmd.String("port"),
				DevMode:    cmd.Bool("dev-mode"),
			}
			return HandleOpenAPI(params)
		},
	}
}

// boolPtr returns a pointer to the given bool value
func boolPtr(val bool) *bool {
	return &val
}