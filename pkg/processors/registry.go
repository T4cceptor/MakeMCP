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
	"fmt"

	"github.com/T4cceptor/MakeMCP/pkg/config"
)

// DefaultRegistry is the global processor registry
var DefaultRegistry = NewRegistry()

// CreateProcessorChain creates a processor chain from configuration
func CreateProcessorChain(configs []config.ProcessorConfig, stage config.ProcessorStage) (*ProcessorChain, error) {
	chain := NewProcessorChain()
	
	for _, cfg := range configs {
		if cfg.Stage != stage {
			continue
		}
		
		processor, exists := DefaultRegistry.Get(cfg.Name)
		if !exists {
			return nil, fmt.Errorf("unknown processor: %s", cfg.Name)
		}
		
		if err := chain.Add(processor, cfg.Config); err != nil {
			return nil, fmt.Errorf("failed to add processor %s: %w", cfg.Name, err)
		}
	}
	
	return chain, nil
}

// ProcessWithChain processes data with a processor chain for a specific stage
func ProcessWithChain(ctx context.Context, configs []config.ProcessorConfig, stage config.ProcessorStage, data *ProcessorData) error {
	chain, err := CreateProcessorChain(configs, stage)
	if err != nil {
		return err
	}
	
	return chain.Process(ctx, data)
}

// GetAvailableProcessors returns all available processor names
func GetAvailableProcessors() []string {
	return DefaultRegistry.List()
}

// GetProcessorsByStage returns all processors for a specific stage
func GetProcessorsByStage(stage config.ProcessorStage) []Processor {
	return DefaultRegistry.GetByStage(stage)
}

// ValidateProcessorConfig validates a processor configuration
func ValidateProcessorConfig(cfg config.ProcessorConfig) error {
	processor, exists := DefaultRegistry.Get(cfg.Name)
	if !exists {
		return fmt.Errorf("unknown processor: %s", cfg.Name)
	}
	
	return processor.ValidateConfig(cfg.Config)
}