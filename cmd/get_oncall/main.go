// Copyright 2024 The Sigstore Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/chainguard-dev/clog"
	"github.com/spf13/cobra"

	"github.com/sigstore/sigstore-devops-tools/cmd"
	"github.com/sigstore/sigstore-devops-tools/pkg/get_oncall"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "get-oncall",
		Short: "Slack slash command to retrieve the information that who is on-call for sigstore infrastructure.",
		Run: func(cmd *cobra.Command, args []string) {
			log := clog.FromContext(cmd.Context())
			o, err := get_oncall.New(cmd.Context())
			if err != nil {
				log.Errorf("failed to create new get_oncall: %v", err)
				os.Exit(1)
			}

			o.StartServer()
		},
	}

	cmd.ExecuteCommand(rootCmd)
}
