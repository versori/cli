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
	"github.com/versori/cli/pkg/cmd/context"
)

func newCtxCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manages the authentications to the Versori platform",
		Long:  "Context sets the configuration for your CLI session.\nIt allows you to manage your authentication tokens and organisations.",
	}

	add := context.NewAdd(c)
	list := context.NewList(c)
	set := context.NewSet(c)
	rm := context.NewRemove(c)

	cmd.AddCommand(add)
	cmd.AddCommand(list)
	cmd.AddCommand(set)
	cmd.AddCommand(rm)

	return cmd
}
