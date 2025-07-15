# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MakeMCP is a Go CLI tool that creates MCP (Model Context Protocol) servers from various sources, enabling AI agents to interact with diverse systems through a unified interface. The tool transforms different types of sources into MCP-compatible servers:

- **OpenAPI specifications** - Convert REST APIs into MCP tools (current implementation)
- **CLI tools** - Generate MCP tools from command-line utilities using their --help descriptions (planned)
- **Frameworks** - Create MCP interfaces from framework documentation and code (planned)
- **MCP server proxying** - Centralize control over multiple MCP servers by proxying them (planned)

The current implementation focuses on OpenAPI specifications as a well-defined starting point, but the architecture is designed to support multiple source types through a modular approach.

## Core Components

### Entry Point
- `makemcp.go` - Main CLI application using urfave/cli/v3
- `app.go` - MCP server initialization and tool registration

### Key Files
- `models.go` - Core data structures (MakeMCPApp, MakeMCPTool, CLIParams, etc.)
- `openapi_mcp.go` - OpenAPI parsing and MCP tool generation logic
- `utils.go` - Utility functions for API client creation and config file handling

### Data Flow
1. CLI parses source specification (OpenAPI spec, CLI help output, framework docs, etc.)
2. Source-specific parser converts operations/commands into MCP tools with proper schemas
3. Handler functions are generated for each tool to interact with the underlying system
4. MCP server is started with stdio or HTTP transport

## Build and Development

### Build Commands
```bash
# Build the binary
go build -o makemcp .

# Run directly
go run . [commands]
```

### Testing the CLI
```bash
# Show help
./makemcp --help

# Test OpenAPI integration (config only)
makemcp openapi -s 'http://localhost:8081/openapi.json' -b "http://localhost:8081" --config-only true
```

## Key Architecture Patterns

### MCP Tool Generation
- Each OpenAPI operation becomes an MCP tool
- Tool schemas are extracted from OpenAPI parameters (path, query, header, cookie, body)
- Handler functions route parameters to appropriate HTTP request locations
- Tools include detailed descriptions with parameter examples

### Parameter Handling
- Parameters are grouped by location using `SplitParams` struct
- Path parameters are substituted in URL templates
- Query parameters are URL-encoded for GET/DELETE requests
- Body parameters are JSON-marshaled for POST/PUT requests
- Headers and cookies are set on HTTP requests

### Transport Support
- Stdio transport for direct MCP protocol communication
- HTTP transport for web-based MCP servers
- Configurable port for HTTP servers

## Dependencies

Core libraries:
- `github.com/urfave/cli/v3` - CLI framework
- `github.com/getkin/kin-openapi` - OpenAPI specification parsing
- `github.com/mark3labs/mcp-go` - MCP protocol implementation

## Configuration

The tool generates `.makemcp.json` configuration files that store:
- MCP server metadata (name, version, transport)
- Tool definitions with schemas and handler information
- OpenAPI configuration (base URL, etc.)

## Common Usage Pattern

1. Point tool at OpenAPI specification URL or file
2. Specify base URL for API calls
3. Choose transport (stdio for MCP clients, http for web access)
4. Tool creates MCP server with auto-generated tools for each API endpoint