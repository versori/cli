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
	"github.com/versori/cli/pkg/cmd/kv"
)

func newKvCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kv",
		Short: "Inspect and manage KV store entries",
		Long: `Inspect and manage entries in a project's KV store.

Read commands (stores list, list, count, get) are safe to run for diagnosis. Mutation commands
(set, delete, wipe) change live workflow state and should only be run when explicitly requested.`,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			c.LoadConfigAndContext()
		},
	}

	cmd.AddCommand(
		kv.NewStores(c),
		kv.NewList(c),
		kv.NewCount(c),
		kv.NewGet(c),
		kv.NewSet(c),
		kv.NewDelete(c),
		kv.NewWipe(c),
	)

	return cmd
}
