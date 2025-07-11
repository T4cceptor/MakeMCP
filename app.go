package main

import (
	"log"

	"github.com/mark3labs/mcp-go/server"
)

func GetMCPServer(app MakeMCPApp) *server.MCPServer {
	mcp_server := server.NewMCPServer(
		app.Name,
		app.Version,
		server.WithToolCapabilities(true),
	)
	for i := range app.Tools {
		tool := &(app.Tools[i])
		mcp_server.AddTool(tool.Tool, tool.HandlerFunction)
		log.Printf("Registered TOOL: %s with func: %#v", tool.Name, tool.HandlerFunction)
	}
	return mcp_server
}
