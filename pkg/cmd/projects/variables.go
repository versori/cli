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
	"github.com/versori/cli/pkg/cmd/projects/variables"
)

// NewVariablesCommand wires `versori projects variables` parent command.
// Manages the project-level DynamicVariablesSchema that declares which keys
// end-user activations on this project may set.
//
// The subcommands split into two tiers:
//   - High-level (list/add/update/remove) work in terms of Name / Type / Description /
//     Required and cover the 95% case without ever exposing JSON Schema to the user.
//   - Low-level (get/set/patch) operate on the raw JSON Schema and are kept as escape
//     hatches for advanced shapes (enum, default, nested object, patternProperties).
func NewVariablesCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "variables",
		Aliases: []string{"variable"},
		Short:   "Manage a project's DynamicVariablesSchema (declares valid activation-variable keys)",
	}

	cmd.AddCommand(variables.NewList(c))
	cmd.AddCommand(variables.NewAdd(c))
	cmd.AddCommand(variables.NewUpdate(c))
	cmd.AddCommand(variables.NewRemove(c))

	cmd.AddCommand(variables.NewGet(c))
	cmd.AddCommand(variables.NewSet(c))

	return cmd
}
