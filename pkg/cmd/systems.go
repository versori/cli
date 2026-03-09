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
	systems "github.com/versori/cli/pkg/cmd/systems"
)

func newSystemsCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "systems",
		Aliases: []string{"system"},
		Short:   "Manage systems",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			c.LoadConfigAndContext()
		},
	}

	systemsList := systems.NewList(c)
	systemsCreate := systems.NewCreate(c)
	systemsAddAuthScheme := systems.NewAddAuthScheme(c)

	// Attach subcommands
	cmd.AddCommand(systemsList)
	cmd.AddCommand(systemsCreate)
	cmd.AddCommand(systemsAddAuthScheme)

	return cmd
}
