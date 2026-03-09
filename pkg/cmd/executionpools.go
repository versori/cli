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
	executionpools "github.com/versori/cli/pkg/cmd/executionpools"
)

func newExecutionPoolsCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "execution-pools",
		Aliases: []string{"ep"},
		Short:   "Manage execution pools",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			c.LoadConfigAndContext()
		},
	}

	listCmd := executionpools.NewList(c)

	cmd.AddCommand(listCmd)

	return cmd
}
