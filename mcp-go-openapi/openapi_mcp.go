package mcpgoopenapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// HTTPRoute represents a parsed OpenAPI HTTP route.
type HTTPRoute struct {
	Method      string
	Path        string
	OperationID string
	Summary     string
	Tags        []string
	// ... add more fields as needed
}

type HTTPClient struct{}
type OpenAPISpec struct {
	Raw map[string]interface{} // Holds the raw OpenAPI JSON structure
}

// NewOpenAPISpec parses a JSON byte slice into an OpenAPISpec
func NewOpenAPISpec(data []byte) (*OpenAPISpec, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &OpenAPISpec{Raw: raw}, nil
}

// ParseOpenAPISpecToRoutes should parse your OpenAPI spec and return HTTPRoute slices.
func ParseOpenAPISpecToRoutes(spec *OpenAPISpec) []HTTPRoute {
	// Implement this function based on your OpenAPI parsing logic.
	return nil
}

// NewFastMCPOpenAPI initializes the MCP server from an OpenAPI spec.
func NewFastMCPOpenAPI(
	spec *OpenAPISpec,
	client *HTTPClient,
	name string,
	version string,
) *server.MCPServer {
	s := server.NewMCPServer(
		name,
		version,
		server.WithToolCapabilities(true),
	)

	routes := ParseOpenAPISpecToRoutes(spec)
	for _, route := range routes {
		tool := mcp.NewTool(
			getToolName(route),
			mcp.WithDescription(route.Summary),
			// Add more argument definitions here based on OpenAPI spec
		)
		s.AddTool(tool, makeOpenAPIToolHandler(route, client))
		log.Printf("Registered TOOL: %s (%s %s)", tool.Name, route.Method, route.Path)
	}
	return s
}

// getToolName generates a tool name from the OpenAPI route.
func getToolName(route HTTPRoute) string {
	if route.OperationID != "" {
		return route.OperationID
	}
	if route.Summary != "" {
		return strings.ReplaceAll(strings.ToLower(route.Summary), " ", "_")
	}
	return fmt.Sprintf("%s_%s", strings.ToLower(route.Method), strings.ReplaceAll(route.Path, "/", "_"))
}

// makeOpenAPIToolHandler returns a handler for the OpenAPI tool.
func makeOpenAPIToolHandler(route HTTPRoute, client *HTTPClient) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// TODO: Map request arguments to HTTP request for the OpenAPI route
		// Use client to make the HTTP request and return the result
		return mcp.NewToolResultText(fmt.Sprintf("Called %s %s", route.Method, route.Path)), nil
	}
}
