package core

import (
	"encoding/json"
	"fmt"
)

// UnmarshalConfigWithTypedParams is a generic helper function that unmarshals a MakeMCPApp
// configuration with both tools and source parameters of specific concrete types, then converts them to interfaces.
func UnmarshalConfigWithTypedParams[T MakeMCPTool, P AppParams](data []byte) (*MakeMCPApp, error) {
	var configData struct {
		Name       string `json:"name"`
		Version    string `json:"version"`
		SourceType string `json:"sourceType"`
		Tools      []T    `json:"tools"`
		AppParams  P      `json:"config"`
	}

	if err := json.Unmarshal(data, &configData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Convert concrete tools to interface slice
	var tools []MakeMCPTool
	for _, tool := range configData.Tools {
		tools = append(tools, tool)
	}

	return &MakeMCPApp{
		Name:       configData.Name,
		Version:    configData.Version,
		SourceType: configData.SourceType,
		Tools:      tools,
		AppParams:  configData.AppParams,
	}, nil
}
