package config

import (
	"encoding/json"
	"fmt"
)

// UnmarshalConfigWithTools is a generic helper function that unmarshals a MakeMCPApp
// configuration with tools of a specific concrete type T, then converts them to the interface.
func UnmarshalConfigWithTools[T MakeMCPTool](data []byte) (*MakeMCPApp, error) {
	var configData struct {
		Name       string    `json:"name"`
		Version    string    `json:"version"`
		SourceType string    `json:"sourceType"`
		Tools      []T       `json:"tools"`
		CliParams  CLIParams `json:"config"`
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
		CliParams:  configData.CliParams,
	}, nil
}
