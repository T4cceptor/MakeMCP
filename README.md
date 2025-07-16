# MakeMCP

**Transform APIs into MCP servers for AI agents**

MakeMCP is a simple CLI tool that converts OpenAPI specifications into MCP (Model Context Protocol) servers. Point it at any REST API with an OpenAPI spec, and it creates a fully compliant MCP server that works with all MCP clients and tools.

## Key Benefits

- **Universal compatibility**: Fully MCP-compliant, works with Claude Desktop, VSCode, and any MCP client
- **Simple usage**: One command transforms any OpenAPI spec into an MCP server
- **No configuration needed**: Just provide an OpenAPI spec URL and base API URL
- **Standard MCP protocol**: Integrates seamlessly with existing MCP ecosystem
- **Multiple transport options**: Supports stdio (for direct integration) and HTTP transports

## Quickstart

### Install

**From source (current):**
```bash
git clone https://github.com/your-org/makemcp
cd makemcp
go build -o makemcp .
```

**Package managers (coming soon):**
```bash
# macOS/Linux
brew install makemcp

# Windows
scoop install makemcp

# Linux
apt-get install makemcp  # Ubuntu/Debian
yum install makemcp      # RHEL/CentOS
```

### Basic Usage

**1. Run as MCP server (stdio transport):**
```bash
makemcp openapi -s "http://localhost:8081/openapi.json" -b "http://localhost:8081"
```

**2. Run as HTTP server:**
```bash
makemcp openapi -s "http://localhost:8081/openapi.json" -b "http://localhost:8081" -t http
```

**3. Use with MCP clients:**

**Claude Desktop:**
```json
{
  "mcpServers": {
    "my-api": {
      "command": "makemcp",
      "args": [
        "openapi",
        "-s", "http://localhost:8081/openapi.json",
        "-b", "http://localhost:8081"
      ]
    }
  }
}
```

**VSCode with MCP extension:**
```json
{
  "my-api": {
    "type": "stdio",
    "command": "makemcp",
    "args": [
      "openapi",
      "-s", "http://localhost:8081/openapi.json",
      "-b", "http://localhost:8081"
    ]
  }
}
```

### Examples

**Using a public API (JSONPlaceholder):**
```bash
makemcp openapi -s "https://jsonplaceholder.typicode.com/openapi.json" -b "https://jsonplaceholder.typicode.com"
```

**Using a local OpenAPI file:**
```bash
makemcp openapi -s "./my-api-spec.json" -b "http://localhost:3000"
```

**Generate configuration only (no server):**
```bash
makemcp openapi -s "http://localhost:8081/openapi.json" -b "http://localhost:8081" --config-only
```


## CLI Reference

### Commands

**`makemcp openapi`** - Convert OpenAPI specification to MCP server

**Options:**
- `-s, --spec <url|file>` - OpenAPI specification URL or file path (required)
- `-b, --base-url <url>` - Base URL for API requests (required)
- `-t, --transport <stdio|http>` - Transport protocol (default: stdio)
- `--config-only` - Generate configuration file only, don't start server
- `--port <port>` - Port for HTTP transport (default: 8080)
- `-h, --help` - Show help

**Examples:**
```bash
# Basic usage
makemcp openapi -s "http://api.example.com/openapi.json" -b "http://api.example.com"

# HTTP transport
makemcp openapi -s "./spec.json" -b "http://localhost:3000" -t http --port 9000

# Configuration only
makemcp openapi -s "http://api.example.com/openapi.json" -b "http://api.example.com" --config-only
```

### Global Options
- `--version` - Show version information
- `--help` - Show help

## Configuration

MakeMCP generates `.makemcp.json` configuration files that contain:
- MCP server metadata (name, version, transport)
- Tool definitions with schemas and handler information
- OpenAPI configuration (base URL, etc.)

**Example configuration:**
```json
{
  "name": "my-api-server",
  "version": "1.0.0",
  "transport": "stdio",
  "tools": [
    {
      "name": "get_users",
      "description": "Retrieve list of users",
      "inputSchema": {
        "type": "object",
        "properties": {
          "limit": {"type": "integer", "description": "Maximum number of users"}
        }
      }
    }
  ]
}
```

## Troubleshooting

**Common Issues:**

**"OpenAPI spec not found"**
- Verify the URL or file path is correct
- Check network connectivity for remote URLs
- Ensure the OpenAPI spec is valid JSON/YAML

**"Connection refused"**
- Verify the base URL is accessible
- Check if the API server is running
- Ensure firewall/network settings allow connections

**"Invalid OpenAPI specification"**
- Validate your OpenAPI spec using tools like Swagger Editor
- Check for syntax errors in JSON/YAML
- Ensure the spec follows OpenAPI 3.0+ format

**MCP client can't connect:**
- Check the command path in your MCP client configuration
- Verify makemcp is installed and accessible
- Check transport settings (stdio vs http)

## Dependencies

- [urfave/cli](https://cli.urfave.org/) - CLI framework
- [kin-openapi](https://github.com/getkin/kin-openapi) - OpenAPI specification parsing
- [mcp-go](https://github.com/mark3labs/mcp-go) - MCP protocol implementation

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Roadmap

**Short term:**
- CLI tool integration (auto-generate MCP tools from `--help` output)
- MCP server proxying capabilities
- Enhanced processors for custom tool modification
- Web frontend for MCP tool management

**Long term:**
- Framework documentation processing
- Advanced authentication mechanisms
- Plugin system for custom processors
- Performance optimizations for large APIs
