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

package projects

import (
	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/projects/systems"
)

// NewSystemsCommand creates the `versori projects systems` parent command and wires its subcommands.
func NewSystemsCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "systems",
		Short: "Manage systems within a project",
	}

	add := systems.NewSystemsAdd(c)
	bootstrap := systems.NewBootstrap(c)
	list := systems.NewSystemsList(c)
	remove := systems.NewSystemsRemove(c)
	deleteTemplate := systems.NewDeleteConnectionTemplate(c)
	updateTemplate := systems.NewUpdateConnectionTemplate(c)
	listConnections := systems.NewListConnections(c)

	cmd.AddCommand(add)
	cmd.AddCommand(bootstrap)
	cmd.AddCommand(list)
	cmd.AddCommand(remove)
	cmd.AddCommand(deleteTemplate)
	cmd.AddCommand(updateTemplate)
	cmd.AddCommand(listConnections)

	return cmd
}
