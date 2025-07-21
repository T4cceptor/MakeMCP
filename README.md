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

### Quick Install

With homebrew (macOS / Linux, note: this requires go installed and in path)
```bash
# Add the tap (once the tap is created)
brew tap T4cceptor/makemcp

# Install MakeMCP
brew install makemcp
```
For latest dev version use: `brew install --HEAD t4cceptor/makemcp/makemcp`

Without homebrew (Windows)
```bash
# Linux/macOS - automatic download and install
curl -sSL https://raw.githubusercontent.com/T4cceptor/MakeMCP/main/scripts/install.sh | bash
```

### Other Installation Methods

**Download Binary:**
Download the latest binary for your platform from [GitHub Releases](https://github.com/T4cceptor/MakeMCP/releases)

**Go Install:**
```bash
go install github.com/T4cceptor/MakeMCP@latest
```

**Build from Source:**
```bash
git clone https://github.com/T4cceptor/MakeMCP
cd MakeMCP
make build
```

### Basic Usage

**1. Run as MCP server (stdio transport):**
```bash
makemcp openapi -s "http://localhost:8081/openapi.json" -b "http://localhost:8081"
```

**2. Run as HTTP server:**
```bash
makemcp openapi -s "http://localhost:8081/openapi.json" -b "http://localhost:8081" -t http -port 3000
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
- `--dev-mode` - Enable development mode (suppresses security warnings)
- `-h, --help` - Show help

**`makemcp load <config-file>`** - Load MakeMCP configuration and start server

**Options:**
- `-t, --transport <stdio|http>` - Override transport protocol from config
- `--port <port>` - Override port from config
- `-h, --help` - Show help

**Examples:**
```bash
# Basic usage
makemcp openapi -s "http://api.example.com/openapi.json" -b "http://api.example.com"

# HTTP transport
makemcp openapi -s "./spec.json" -b "http://localhost:3000" -t http --port 9000

# Configuration only
makemcp openapi -s "http://api.example.com/openapi.json" -b "http://api.example.com" --config-only

# Load and run from saved configuration
makemcp load makemcp.json

# Load configuration with overrides
makemcp load makemcp.json --transport http --port 9090
```

### Global Options
- `--version` - Show version information
- `--help` - Show help

## Configuration

MakeMCP generates `makemcp.json` configuration files that contain:
- MCP server metadata (name, version, transport)
- Tool definitions with schemas and handler information
- OpenAPI source configuration (base URL, spec location, etc.)

Use `--config-only` to generate configuration without starting the server, then use `makemcp load` to start from the saved configuration.

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

**Common Steps:**

- for OpenAPI:
  - if server start fails:
    - this is likely due to a configuration and/or parsing issue from source
    - Verify both URL/path to OpenAPI specification json is correct and publicly available (we do not support auth yet)
    - Verify that the specification is valid (e.g. if you provide a file check if its indeed a valid OpenAPI spec) and it follows OpenAPI 3.0+ format - older formats are currently not supported
  - if tool invocation fails:
    - check how parameters are provided to the MCP tool, the correct format should be something like `<parameter_location>__<parameter_name>: <parameter value>`, for example: `body__user_email: test@example.com` or `path__user_id: 123` - the location is delimited using double underscores `__` which is then used for parsing the request
    - Ensure firewall/network settings allow connections
    - Check transport settings (stdio vs http)

## Dependencies

- [urfave/cli](https://cli.urfave.org/) - CLI framework
- [kin-openapi](https://github.com/getkin/kin-openapi) - OpenAPI specification parsing
- [mcp-go](https://github.com/mark3labs/mcp-go) - MCP protocol implementation

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Features

**Current:**
- ✅ OpenAPI 3.0+ specification parsing
- ✅ Full MCP protocol compliance
- ✅ Multiple transport support (stdio, HTTP)
- ✅ Comprehensive parameter handling (path, query, body, headers)
- ✅ Configuration file generation and loading
- ✅ Cross-platform binaries (Linux, macOS, Windows, FreeBSD)
- ✅ Security features (SSRF protection, URL validation)
- ✅ Development mode for testing

**Planned:**
- 🔄 CLI tool integration (auto-generate MCP tools from `--help` output)
- 🔄 Package manager support (Homebrew, APT, etc.)
- 🔄 GraphQL specification support
- 🔄 MCP server proxying capabilities
- 🔄 Enhanced authentication mechanisms
- 🔄 Plugin system for custom processors
