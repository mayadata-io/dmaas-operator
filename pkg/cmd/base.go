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

// Package cmd provides function for command utility
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCommand returns the command for dmaas-operator
func NewCommand(name string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: "DMaaS operator to backup and restore k8s resources",
	}

	config := NewConfig()

	cmd.AddCommand(
		NewCmdVersion(),
		NewCmdServer(config),
	)

	return cmd
}

// CheckError verify the given error
func CheckError(err error) {
	if err != nil {
		fmt.Printf("Error occurred: %v\n", err)
		os.Exit(1)
	}
}
