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

package notifications

import (
	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/notifications/channels"
)

// NewChannelsCommand creates the `versori notifications channels` parent command and wires its subcommands.
func NewChannelsCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channels",
		Aliases: []string{"channel"},
		Short:   "Manage organisation-wide notification channels",
	}

	cmd.AddCommand(channels.NewList(c))
	cmd.AddCommand(channels.NewCreate(c))
	cmd.AddCommand(channels.NewDelete(c))

	return cmd
}
