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

package sources

import (
	core "github.com/T4cceptor/MakeMCP/pkg/core"
	"github.com/urfave/cli/v3"
)

// MakeMCPSource defines the interface for all MCP source implementations.
type MakeMCPSource interface {
	// Name returns the name of the source type
	Name() string

	// ParseParams converts raw CLI input into typed source-specific parameters
	ParseParams(input *core.CLIParamsInput) (core.SourceParams, error)

	// Parse creates a MakeMCPApp configuration from typed parameters (Step 1: no handlers)
	Parse(params core.SourceParams) (*core.MakeMCPApp, error)

	// AttachToolHandlers adds tool handler functions to an existing MakeMCPApp (Step 2: ready to serve)
	AttachToolHandlers(app *core.MakeMCPApp) error

	UnmarshalConfig(data []byte) (*core.MakeMCPApp, error)

	// GetCommand returns the CLI command for this source
	GetCommand() *cli.Command
}
