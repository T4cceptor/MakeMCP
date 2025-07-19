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

### Makefile Commands
The project includes a comprehensive Makefile with automation for common tasks:

**Primary Commands:**
```bash
# Build the binary (recommended)
make build

# Build and run with help
make run

# Clean build artifacts
make clean

# Run tests
make test

# Full development workflow (clean, format, vet, test, build)
make dev
```

**Code Quality:**
```bash
# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Run go vet
make vet

# Tidy dependencies
make tidy

# Install development dependencies
make dev-deps
```

**Cross-Platform Building:**
```bash
# Build for all platforms (Linux, macOS, Windows)
make build-all

# Prepare release builds
make release
```

**Testing Commands:**
```bash
# Test config generation with local server
make local-config-test

# Alias for local-config-test
make local-test
```

### Manual Build Commands
```bash
# Build the binary manually
go build -o makemcp cmd/makemcp.go

# Run directly
go run cmd/makemcp.go [commands]
```

### Testing the CLI
```bash
# Show help
./build/makemcp --help

# Test OpenAPI integration (config only) - use Makefile command
make local-test

# Manual test of OpenAPI integration
./build/makemcp openapi -s 'http://localhost:8081/openapi.json' -b "http://localhost:8081" --config-only true
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