# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MakeMCP is a Go CLI tool that creates MCP (Model Context Protocol) servers from various sources, enabling AI agents to interact with diverse systems through a unified interface. The tool transforms different types of sources into MCP-compatible servers:

- **OpenAPI specifications** - Convert REST APIs into MCP tools (current implementation)
- **CLI tools** - Generate MCP tools from command-line utilities using their --help descriptions (planned)
- **Frameworks** - Create MCP interfaces from framework documentation and code (planned)
- **MCP server proxying** - Centralize control over multiple MCP servers by proxying them (planned)

The current implementation focuses on OpenAPI specifications as a well-defined starting point, but the architecture is designed to support multiple source types through a modular approach.

## Project Structure

### Entry Point
- `cmd/makemcp.go` - Main CLI application using urfave/cli/v3

### Core Packages
- `internal/` - Internal application logic
  - `commands.go` - CLI command definitions and handlers
  - `loader.go` - Configuration loading and management
  - `orchestrator.go` - MCP server orchestration
  - `server.go` - MCP server implementation
- `pkg/core/` - Core MCP functionality
  - `app.go` - MCP application structure
  - `mcp.go` - MCP protocol implementation
  - `params.go` - Parameter handling utilities
  - `utils.go` - General utility functions
- `pkg/sources/` - Source type implementations
  - `interface.go` - Source interface definition
  - `openapi/` - OpenAPI source implementation
    - `command.go` - CLI command for OpenAPI
    - `mcp.go` - MCP tool generation from OpenAPI
    - `models.go` - OpenAPI-specific data structures
    - `params.go` - Parameter handling for OpenAPI
    - `utils.go` - OpenAPI utility functions
  - `registry.go` - Source type registry

### Scripts and Tools
- `scripts/` - Automation and utility scripts
  - `install.sh` - Installation script for releases
  - `tag-release.sh` - Automated release tagging script

### Testing Infrastructure
- `testbed/openapi/` - OpenAPI integration tests
  - `sample_specifications/` - Test OpenAPI specifications
  - `expected_results/` - Expected MCP configurations
  - `results/` - Generated test results
  - `integration_test.go` - Integration test suite

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
# Test OpenAPI integration with local server
make local-openapi-test

# Alias for local-openapi-test  
make local-test

# Test loading from config file
make local-file-test

# Run integration tests
make integration-test
```

**Release Commands:**
```bash
# Tag a new release (interactive)
make tag-release          # defaults to patch
make tag-release major    # major version bump
make tag-release minor    # minor version bump
make tag-release patch    # patch version bump
```

### Manual Build Commands
```bash
# Build the binary manually
go build -o build/makemcp cmd/makemcp.go

# Run directly
go run cmd/makemcp.go [commands]

# Build with version info
go build -ldflags "-s -w -X main.version=v1.0.0" -o build/makemcp cmd/makemcp.go
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
- Parameters are grouped by location using `ToolParams` struct
- Path parameters are substituted in URL templates
- Query parameters are URL-encoded for GET/DELETE requests
- Body parameters are JSON-marshaled for POST/PUT requests
- Headers and cookies are set on HTTP requests
- Prefix-based parameter parsing (e.g., `path__id`, `query__limit`, `header__auth`)

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

The tool generates `makemcp.json` configuration files that store:
- MCP server metadata (name, version, transport)
- Tool definitions with schemas and handler information
- Source-specific configuration (OpenAPI base URL, spec location, etc.)
- Generated tool schemas and parameter mappings

Configuration files can be:
- Generated with `--config-only` flag
- Loaded and executed with `makemcp load <config-file>`
- Customized for different environments and use cases

## Common Usage Pattern

1. Point tool at OpenAPI specification URL or file
2. Specify base URL for API calls
3. Choose transport (stdio for MCP clients, http for web access)
4. Tool creates MCP server with auto-generated tools for each API endpoint

## Recent Major Improvements (2025-07-21)

### Code Quality and Linting
The project now has comprehensive linting configuration and all critical issues have been resolved:

**Lint Command:**
- `make lint` now passes successfully with comprehensive golangci-lint configuration
- Critical issues fixed: cyclomatic complexity, unused parameters, error handling
- Style issues appropriately ignored via `.golangci.yml` exclude rules
- Compatible with CI/CD pipelines

**Code Improvements:**
- **Reduced Cyclomatic Complexity**: Refactored `GetHandlerFunction` from complexity 17 to ~8 by extracting helper functions:
  - `buildRequestURL()` - URL construction and query parameters
  - `buildRequestBody()` - Request body preparation for non-GET methods  
  - `setRequestHeaders()` - Headers and cookies application
- **Fixed Error Handling**: All `errcheck` violations resolved with proper error checking
- **Removed Dead Code**: Eliminated unused parameters and functions
- **Modern Go Practices**: Updated octal literals (`0644` → `0o644`), improved type safety

### Build System Fixes
**GitHub Actions:**
- Fixed build failures in CI/CD pipeline by correcting Go build paths
- Updated both `ci.yml` and `release.yml` to use `./cmd/makemcp.go` instead of `.`
- Cross-platform builds now work correctly for all target platforms

**Makefile Improvements:**
- Fixed broken `local-test` target dependency
- All make commands verified and working:
  - `make dev` - Complete development workflow
  - `make lint` - Passes with proper ignore rules
  - `make test` - Full test suite (39/39 tests passing)
  - `make build` - Binary compilation
  - `make local-test` - OpenAPI integration testing

### Architecture Enhancements
**Parameter Handling:**
- Improved `ToolParams` struct usage for better type safety
- Enhanced parameter parsing with prefix-based approach
- Better separation of concerns between URL, body, and header handling

**Error Handling:**
- Proper file closing with error checking
- Improved error propagation through the application
- Better logging for debugging

### Testing and Reliability
**Test Coverage:**
- All tests passing with race condition detection (`-race` flag)
- Comprehensive test coverage for core functionality
- Integration tests for OpenAPI source parsing
- Proper test cleanup and isolation

**Development Workflow:**
- `make dev` provides complete CI-like workflow locally
- Consistent formatting with `make fmt`
- Dependency verification with `make tidy`
- Development dependencies managed with `make dev-deps`

### Configuration and Documentation
**Lint Configuration (`.golangci.yml`):**
- Comprehensive linter configuration with 23 enabled linters
- Appropriate exclusions for cosmetic issues (godot, naming suggestions)
- Test-specific exclusions for appropriate linters
- 5-minute timeout for complex analysis

**Build Configuration:**
- Consistent build flags across Makefile and GitHub Actions
- Version injection working correctly
- Cross-platform compilation support
- Proper binary naming and output directories

### Best Practices Implementation
**Code Style:**
- Modern Go idioms throughout codebase
- Consistent error handling patterns
- Proper resource cleanup (file handles, HTTP responses)
- Type safety improvements with pointer receivers for large structs

**Development Process:**
- Lint-first development approach
- Comprehensive testing before commits
- Makefile-driven development workflow
- CI/CD pipeline compatibility

## Development Workflow Recommendations

**For New Features:**
1. Run `make dev` to ensure clean starting state
2. Implement feature with proper error handling
3. Add tests for new functionality
4. Verify `make lint` passes
5. Test with `make local-test` if OpenAPI-related

**For Bug Fixes:**
1. Write failing test first
2. Implement fix
3. Ensure `make dev` completes successfully
4. Verify fix doesn't break existing functionality

**Before Commits:**
- Always run `make lint` to ensure code quality
- Run `make test` to verify no regressions
- Use `make build` to ensure compilation succeeds

**Release Process:**
1. Ensure all changes are committed and pushed
2. Run `make tag-release [major|minor|patch]` to create and push a new version tag
3. GitHub Actions will automatically build and publish the release
4. Verify release artifacts are published correctly

## Scripts and Automation

### Release Management
- `scripts/tag-release.sh` - Automated version tagging with semantic versioning
  - Automatically determines next version from existing git tags
  - Supports major, minor, and patch version bumps
  - Creates annotated git tags and pushes to trigger releases
  - Interactive confirmation before creating tags

### Installation
- `scripts/install.sh` - Cross-platform installation script
  - Downloads latest release for user's platform
  - Installs to appropriate system location
  - Handles permissions and PATH configuration

## Integration Testing

The project includes comprehensive integration tests in `testbed/openapi/`:

**Test Specifications:**
- Adobe AEM API
- AWS S3 API
- FastAPI sample
- GoFuego framework
- Salesforce API
- Simple API with body parameters

**Test Process:**
1. Parse OpenAPI specifications from `sample_specifications/`
2. Generate MCP configurations
3. Compare against expected results in `expected_results/`
4. Validate tool schemas and parameter handling
5. Ensure backwards compatibility

**Running Tests:**
```bash
# Run all integration tests
make integration-test

# Run specific test categories
cd testbed/openapi && go test -v
```