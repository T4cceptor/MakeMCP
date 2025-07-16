package main

import (
	"context"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
)

type TransportType string

const (
	TransportTypeHTTP  TransportType = "http"
	TransportTypeStdio TransportType = "stdio"
)

// APIClient struct to encapsulate baseURL and http.Client
type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

/*
OpenAPI:
- base url
- endpoints
	-> endpoints have to provide all necessary information to actually call the base url + path
	-> method + path + params
	-> params (incl. body, headers, query params, and path params)
	have to be provided by the AI when calling the endpoint
*/

type MakeMCPProcessor struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
	// TODO: check if this is appropriate or if we should use a list instead
}

// Data provided to create the corresponding operation in MakeMCP
type ToolSource struct {
	// source of the tool handler data
	URI string `json:"name"`
	// actual input data used to create the MakeMCP config
	Data []byte `json:"data"`
}

// Intended to be used as a potential handler input
// Defines how a particular endpoint is to be called
type OpenAPIHandlerInput struct {
	Method     string
	Path       string
	Headers    map[string]string
	Cookies    map[string]string
	BodyAppend map[string]any
}

func NewOpenAPIHandlerInput(method, path string) OpenAPIHandlerInput {
	return OpenAPIHandlerInput{
		Method:     method,
		Path:       path,
		Headers:    make(map[string]string),
		Cookies:    make(map[string]string),
		BodyAppend: make(map[string]any),
	}
}

// Extends mcp.Tool with additional MakeMCP information.
type MakeMCPTool struct {
	// will be provided to tool handler function as-is,
	// the tool handler needs to unmarshall this properly
	// TODO: explain this better!
	// TODO: how will this be used?
	// TODO: to make this useful the handler function needs to be able to
	// process it in some pre-defined way
	HandlerInput        map[string]any       `json:"handlerInput,omitempty"`
	OpenAPIHandlerInput *OpenAPIHandlerInput `json:"oapiHandlerInput,omitempty"`

	HandlerFunction func(
		ctx context.Context,
		request mcp.CallToolRequest,
	) (*mcp.CallToolResult, error) `json:"-"`

	// Defines processors (incl their handler functions) that are applied
	// to each result of the MCP call in order
	Processors []MakeMCPProcessor `json:"processors,omitempty"`

	// holds the original data provided when creating the config
	ToolSource ToolSource `json:"toolSource"`

	mcp.Tool
}

type OpenAPIConfig struct {
	BaseUrl string
}

// Struct to hold all information about the MCP server, follows the MCP protocol.
type MakeMCPApp struct {
	Name    string        `json:"name"`
	Version string        `json:"version"`
	Tools   []MakeMCPTool `json:"tools"`
	// TODO: support resources, resource templates, and prompts

	Transport string  `json:"transport"`
	Port      *string `json:"port,omitempty"`

	// non-MCP fields
	// holds the original OpenAPI config - TBD
	OpenAPIConfig *OpenAPIConfig `json:"openapiConfig,omitempty"`
}

func NewMakeMCPApp(name, version, transport string) MakeMCPApp {
	return MakeMCPApp{
		name,
		version,
		[]MakeMCPTool{}, // empty slice
		string(transport),
		nil,
		nil,
	}
}

// The root json holds all apps and their configs as key-value pairs
type MakeMCPApps map[string]MakeMCPApp

// ToolInputProperty defines a property in the input schema for an MCP tool.
type ToolInputProperty struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location"` // OpenAPI 'in' value: path, query, header, cookie, body, etc.
}

// CLIParams holds all CLI parameters for the makemcp openapi action.
type CLIParams struct {
	Specs      string
	BaseURL    string
	Transport  TransportType
	ConfigOnly bool
	Port       string
	DevMode    bool
}

// SplitParams groups all parameter maps by location for handler logic
// Each field is a map[string]interface{} for its respective location
// Used in GetHandlerFunction for robust parameter routing
type SplitParams struct {
	Path   map[string]interface{}
	Query  map[string]interface{}
	Header map[string]interface{}
	Cookie map[string]interface{}
	Body   map[string]interface{}
}

// NewSplitParams returns a SplitParams struct with all maps initialized
func NewSplitParams() SplitParams {
	return SplitParams{
		Path:   map[string]interface{}{},
		Query:  map[string]interface{}{},
		Header: map[string]interface{}{},
		Cookie: map[string]interface{}{},
		Body:   map[string]interface{}{},
	}
}

// AttachParams takes paramList and attaches values to the correct SplitParams fields
func (params *SplitParams) AttachParams(paramList []map[string]interface{}) {
	for _, param := range paramList {
		name, _ := param["parameter_name"].(string)
		value := param["parameter_value"]
		location, _ := param["location"].(string)
		switch location {
		case "path":
			params.Path[name] = value
		case "query":
			params.Query[name] = value
		case "header":
			params.Header[name] = value
		case "cookie":
			params.Cookie[name] = value
		case "body":
			params.Body[name] = value
		default:
			params.Query[name] = value // fallback
		}
	}
}
