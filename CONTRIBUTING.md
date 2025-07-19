# Contributing to MakeMCP

Thank you for your interest in contributing to MakeMCP! This document provides guidelines and information for contributors.

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code. Please see [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for details.

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Git

### Setting up the development environment

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/MakeMCP.git
   cd MakeMCP
   ```
3. Install dependencies:
   ```bash
   go mod download
   ```
4. Build the project:
   ```bash
   go build -o makemcp .
   ```
5. Run tests:
   ```bash
   go test ./...
   ```

## Contribution Process

### Reporting Issues

- Use GitHub Issues to report bugs or request features
- Check existing issues before creating new ones
- Provide clear descriptions with steps to reproduce
- Include relevant system information (OS, Go version, etc.)

### Making Changes

1. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our coding standards

3. **Test your changes**:
   ```bash
   go test ./...
   go build -o makemcp .
   ```

4. **Test with real OpenAPI specs** to ensure functionality

5. **Commit your changes** with clear commit messages:
   ```bash
   git commit -m "Add feature: description of what you added"
   ```

### Pull Request Guidelines

- Create pull requests against the `main` branch
- Provide a clear description of changes
- Reference related issues using `#issue-number`
- Ensure all tests pass
- Follow the existing code style
- Add tests for new functionality
- Update documentation if needed

## Coding Standards

### Go Code Style

- Follow standard Go formatting with `gofmt`
- Use `golint` and `go vet` for code quality
- Follow Go naming conventions
- Write clear, self-documenting code
- Add comments for complex logic

### Code Organization

- Keep functions focused and small
- Use meaningful variable and function names
- Group related functionality in appropriate files
- Maintain separation between CLI, OpenAPI processing, and MCP logic

## Testing Guidelines

### Test Requirements

- Add unit tests for new functions
- Test error conditions and edge cases
- Include integration tests for OpenAPI processing
- Test with various OpenAPI specifications
- Verify MCP protocol compliance

### Testing with MCP Clients

When testing changes:
- Test with stdio transport using Claude Desktop or VSCode
- Test with HTTP transport using web browsers
- Verify tool schemas are correctly generated
- Test parameter handling (path, query, body, headers)

## Adding New Features

### New Source Types

When adding support for new source types (CLI tools, frameworks):
1. Create new parser in separate file (e.g., `cli_mcp.go`)
2. Implement common interfaces from `models.go`
3. Add CLI commands in `makemcp.go`
4. Update documentation and examples

### OpenAPI Enhancements

- Maintain backward compatibility
- Follow OpenAPI 3.0+ specifications
- Test with various real-world API specifications
- Consider edge cases and malformed specs

## Documentation

- Update README.md for user-facing changes
- Update CHANGELOG.md for all changes
- Add inline code comments for complex logic
- Include examples for new features

## Release Process

Maintainers handle releases, but contributors should:
- Update CHANGELOG.md with their changes
- Ensure version compatibility
- Test cross-platform functionality when possible

## Getting Help

- Check existing documentation first
- Search GitHub Issues for similar questions
- Create an issue for questions or problems
- Be patient and respectful in all interactions

## Recognition

Contributors will be recognized in:
- GitHub contributors list
- Release notes for significant contributions
- Project documentation for major features

Thank you for contributing to MakeMCP and helping make AI agent integration easier for everyone!