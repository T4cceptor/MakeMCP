# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-07-16

### Added
- Initial release of MakeMCP CLI tool
- Automatic MCP tool generation from OpenAPI operations
- Parameter handling for path, query, header, cookie, and body parameters
- Support for stdio and HTTP transport protocols
- JSON configuration file generation (`.makemcp.json`)
- Configuration-only mode for generating MCP tool definitions without starting server
- Support for both remote OpenAPI specification URLs and local files

### Features
- **OpenAPI Integration**: Convert REST APIs with OpenAPI 3.0+ specifications into MCP tools
- **Universal MCP Compatibility**: Fully compliant with MCP protocol, works with all MCP clients
- **Flexible Transport**: Choose between stdio (direct integration) or HTTP (web access) transport protocols
- **Auto-generated Tools**: Each OpenAPI operation becomes a properly formatted MCP tool with schemas
- **Parameter Mapping**: Intelligent parameter handling for different HTTP request locations
- **Configuration Export**: Generate reusable configuration files for MCP tool definitions

### Dependencies
- Go 1.21+
- `github.com/urfave/cli/v3` - CLI framework
- `github.com/getkin/kin-openapi` - OpenAPI specification parsing
- `github.com/mark3labs/mcp-go` - MCP protocol implementation

### Known Limitations
- Only OpenAPI specifications are supported (CLI tools and other sources planned for future releases)
- No built-in authentication mechanisms (uses raw API endpoints)
- HTTP transport port is not configurable (fixed to 8080)
- Limited error handling for malformed OpenAPI specifications

