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
	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/issues"
)

func newIssuesCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "issues",
		Aliases: []string{"issue"},
		Short:   "List, inspect, and update issues",
		Long: `Inspect and manage issues raised in the organisation.

Read commands (list, get) are safe to run for diagnosis. The update command changes live issue
state (e.g. acknowledging or resolving) and should only be run when explicitly requested.`,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			c.LoadConfigAndContext()
		},
	}

	cmd.AddCommand(
		issues.NewList(c),
		issues.NewGet(c),
		issues.NewUpdate(c),
	)

	return cmd
}
