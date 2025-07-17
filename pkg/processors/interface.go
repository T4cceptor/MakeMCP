// Copyright 2025 MakeMCP Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processors

import (
	"context"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/T4cceptor/MakeMCP/pkg/config"
)

// ProcessorData contains the data passed between processors
type ProcessorData struct {
	// Request data
	Request     *mcp.CallToolRequest
	Tool        *config.MakeMCPTool
	HTTPRequest *http.Request
	
	// Response data
	HTTPResponse *http.Response
	Result       *mcp.CallToolResult
	
	// Metadata
	Metadata map[string]interface{}
}

// Processor defines the interface for all processor implementations
type Processor interface {
	// Name returns the processor name
	Name() string
	
	// Stage returns the processing stage
	Stage() config.ProcessorStage
	
	// Process executes the processor logic
	Process(ctx context.Context, data *ProcessorData) error
	
	// GetDefaultConfig returns default configuration
	GetDefaultConfig() map[string]interface{}
	
	// ValidateConfig validates processor configuration
	ValidateConfig(cfg map[string]interface{}) error
}

// Registry manages available processors
type Registry struct {
	processors map[string]Processor
}

// NewRegistry creates a new processor registry
func NewRegistry() *Registry {
	return &Registry{
		processors: make(map[string]Processor),
	}
}

// Register adds a processor to the registry
func (r *Registry) Register(processor Processor) {
	r.processors[processor.Name()] = processor
}

// Get retrieves a processor by name
func (r *Registry) Get(name string) (Processor, bool) {
	processor, exists := r.processors[name]
	return processor, exists
}

// List returns all available processor names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.processors))
	for name := range r.processors {
		names = append(names, name)
	}
	return names
}

// GetByStage returns all processors for a specific stage
func (r *Registry) GetByStage(stage config.ProcessorStage) []Processor {
	var processors []Processor
	for _, processor := range r.processors {
		if processor.Stage() == stage {
			processors = append(processors, processor)
		}
	}
	return processors
}

// ProcessorChain manages a chain of processors for a specific stage
type ProcessorChain struct {
	processors []Processor
	configs    []map[string]interface{}
}

// NewProcessorChain creates a new processor chain
func NewProcessorChain() *ProcessorChain {
	return &ProcessorChain{
		processors: make([]Processor, 0),
		configs:    make([]map[string]interface{}, 0),
	}
}

// Add adds a processor to the chain with its configuration
func (pc *ProcessorChain) Add(processor Processor, config map[string]interface{}) error {
	// Validate configuration
	if err := processor.ValidateConfig(config); err != nil {
		return err
	}
	
	pc.processors = append(pc.processors, processor)
	pc.configs = append(pc.configs, config)
	return nil
}

// Process executes all processors in the chain
func (pc *ProcessorChain) Process(ctx context.Context, data *ProcessorData) error {
	for i, processor := range pc.processors {
		// Set processor-specific configuration in metadata
		if data.Metadata == nil {
			data.Metadata = make(map[string]interface{})
		}
		data.Metadata["processor_config"] = pc.configs[i]
		
		if err := processor.Process(ctx, data); err != nil {
			return err
		}
	}
	return nil
}