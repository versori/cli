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
	"github.com/versori/cli/pkg/cmd/projects/users"
)

// NewUsersCommand wires `versori projects users` parent command
func NewUsersCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage users (activations) within a project",
	}

	list := users.NewUsersList(c)
	details := users.NewDetailsActivation(c)

	cmd.AddCommand(list)
	cmd.AddCommand(details)

	return cmd
}
