package main

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func load_openapi_spec(open_api_spec_location string) *openapi3.T {
	loader := openapi3.NewLoader()
	u, err := url.Parse(open_api_spec_location)
	fmt.Printf("%#v", u)
	if err != nil {
		log.Fatal(err)
	}
	doc, err := loader.LoadFromURI(u)
	// doc, err := loader.LoadFromFile("my-openapi-spec.json")
	if err != nil {
		log.Fatal(err)
	}
	return doc
}

func Setup(args MakeConfig) {
	fmt.Println("Starting mcp server...")

	openapi_spec := load_openapi_spec(args.source_uri)
	fmt.Printf("\n\n### OpenAPI doc: %#v \n\n", openapi_spec)

	j := openapi_spec.Paths.Map()
	// fmt.Printf("\n\n###\n Paths: %#v\n", j)
	fmt.Println("Paths:")
	var path_item *openapi3.PathItem
	for k, v := range j {
		fmt.Printf("- %#v: %#v\n", k, v)
		if k == "/" {
			path_item = v
		}
	}

	fmt.Printf("\n %#v \n", path_item)
	fmt.Printf("\n %#v \n", path_item.Get.Description)

	// Create a new MCP server
	s := server.NewMCPServer(
		"Demo 🚀",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Add tool
	tool := mcp.NewTool("hello_world",
		mcp.WithDescription("Say hello to someone"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the person to greet"),
		),
	)

	// Add tool handler
	s.AddTool(tool, helloHandler)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func helloHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
}
