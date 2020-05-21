/*
Copyright 2020 The MayaData Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    https://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version represent current version of dmaas-operator, set during build
	Version string

	// GitSHA represent the latest commit from which dmaas-operator built, set during build
	GitSHA string

	// GitTreeState represent state of git tree, either "clean" or "dirty"
	GitTreeState string
)

// NewCmdVersion creates the version command
func NewCmdVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show dmaas-operator version",
		Run: func(cmd *cobra.Command, args []string) {
			showVersion()
		},
	}

	return cmd
}

func showVersion() {
	fmt.Printf("GitVersion: %s\n", Version)
	fmt.Printf("Gitcommit: %s\n", GitSHA)
	fmt.Printf("GitTreeState: %s\n", GitTreeState)

	fmt.Printf("GO Version: %s\n", runtime.Version())
	fmt.Printf("GO ARCH: %s\n", runtime.GOARCH)
	fmt.Printf("GO OS: %s\n", runtime.GOOS)
}
