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
	"github.com/versori/cli/pkg/cmd/notifications/project"
)

// NewProjectCommand creates the `versori notifications project` parent command and wires its subcommands.
// These commands manage which notification channels alert for a given project + environment.
func NewProjectCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Link, unlink, and list notification channels on a project + environment",
	}

	cmd.AddCommand(project.NewList(c))
	cmd.AddCommand(project.NewLink(c))
	cmd.AddCommand(project.NewUnlink(c))

	return cmd
}
