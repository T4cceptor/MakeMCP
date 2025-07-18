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

package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	internal "github.com/T4cceptor/MakeMCP/internal"
)

// version is set by build flags during release
var version = "dev"

func main() {
	// Initialize registries
	// Todo: refactor this to be in the /sources folder to keep responsibilities seperate
	internal.InitializeRegistries()

	// Create CLI app
	app := &cli.Command{
		Name:     "makemcp",
		Usage:    "Create an MCP server out of anything.",
		Version:  version,
		Commands: internal.GetCommands(),
	}

	// Run the CLI app
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
