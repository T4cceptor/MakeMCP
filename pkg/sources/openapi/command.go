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

package openapi

import (
	"github.com/urfave/cli/v3"
)

var specsFlag cli.StringFlag = cli.StringFlag{
	Name:    "specs",
	Aliases: []string{"s"},
	Value:   "",
	Usage:   "Where to find the OpenAPI specification - can be either a properly formed URL, including protocol, or a file path to a JSON file.",
}

var baseUrlFlag cli.StringFlag = cli.StringFlag{
	Name:    "base-url",
	Aliases: []string{"b"},
	Value:   "",
	Usage:   "Base URL of the OpenAPI specified API. This will be called when invoking the tools.",
}

var timeoutFlag cli.IntFlag = cli.IntFlag{
	Name:    "timeout",
	Aliases: []string{"to"},
	Value:   30,
	Usage:   "HTTP timeout in seconds for API requests (default: 30)",
}

// GetCommand returns the CLI command for this source
func (s *OpenAPISource) GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "openapi",
		Usage: "Use OpenAPI specifications to launch an MCP server locally.",
		Flags: []cli.Flag{
			&specsFlag,
			&baseUrlFlag,
			&timeoutFlag,
		},
	}
}
