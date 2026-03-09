/*
 * Copyright (c) 2026 Versori Group Inc
 *
 * Use of this software is governed by the Business Source License 1.1
 * included in the LICENSE file at the root of this repository.
 *
 * Change Date: 2030-03-01
 * Change License: Apache License, Version 2.0
 *
 * As of the Change Date, in accordance with the Business Source License,
 * use of this software will be governed by the Apache License, Version 2.0.
 */

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "0.1.1"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show versori version",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Flags().BoolP("short", "s", false, "Show short version")

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), version)
	},
}
