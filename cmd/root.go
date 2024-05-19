// Copyright 2024 The Sigstore Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"

	"github.com/chainguard-dev/clog"

	"github.com/spf13/cobra"
)

func ExecuteCommand(rootCmd *cobra.Command) {
	if err := rootCmd.Execute(); err != nil {
		clog.ErrorContext(rootCmd.Context(), err.Error())
		os.Exit(1)
	}
}
