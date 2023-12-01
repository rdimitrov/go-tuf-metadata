// Copyright 2022-2023 VMware, Inc.
//
// This product is licensed to you under the BSD-2 license (the "License").
// You may not use this product except in compliance with the BSD-2 License.
// This product may include a number of subcomponents with separate copyright
// notices and license terms. Your use of these subcomponents is subject to
// the terms and conditions of the subcomponent's license, as noted in the
// LICENSE file.
//
// SPDX-License-Identifier: BSD-2-Clause

package cmd

import (
	"fmt"
	stdlog "log"
	"os"

	"github.com/go-logr/stdr"
	"github.com/rdimitrov/go-tuf-metadata/examples"
	"github.com/rdimitrov/go-tuf-metadata/metadata"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:     "init",
	Aliases: []string{"i"},
	Short:   "Initialize a repository",
	Args:    cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return InitializeCmd()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func InitializeCmd() error {
	// set logger and debug verbosity level
	metadata.SetLogger(examples.WLogger{
		stdr.New(stdlog.New(os.Stdout, "ini_cmd", stdlog.LstdFlags)),
	})
	if Verbosity {
		stdr.SetVerbosity(5)
	}

	fmt.Println("Initialization successful")

	return nil
}
