# MakeMCP

## Features
- transform common APIs into MCP servers, ready to be used by your agents
- create, modify, and manage MCP resources and tools an agent can use
- manage user permissions for MCP actions either by
- leverages more AI-tailored tool description and schemas, to support more complex APIs and use-cases

## Quickstart

### Install
TODO - provide an easy way to install using brew, grep, and other package managers (apt-get)

### How to use
Basically you can run this locally or remote and reference it in your favorite MCP client:

- VSCode:
```json
"my-new-mcp": {
    "type": "stdio",
    "command": "makemcp", // Note: you might need to provide the absolute path where the tool is installed
    "args": [
        "openapi",
        "-s",
        "http://localhost:8081/openapi.json", // path to OpenAPI JSON
        "-b",
        "http://localhost:8081" // base path of the API
    ]
}
```

- based on OpenAPI
    - determine the base URL of the API, and the API spec or have a json file ready
    - run `makemcp openapi -s <url to spec OR path to json file> -b <base url>`
    - this starts a MCP server using stdio transport, for http use `-t http` argument
- based on CLI-tool
    - coming soon


### Build dependencies

- urfave/cli (https://cli.urfave.org/)
- kin-openapi (https://github.com/getkin/kin-openapi)
- mcp-go (https://github.com/mark3labs/mcp-go)

