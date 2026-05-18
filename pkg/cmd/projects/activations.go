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

// NewActivationsCommand wires `versori projects activations` parent command.
// This is a discoverability alias over the activation operations that physically
// live under `versori projects users` — the activation lifecycle is the primary
// thing most people are looking for, so we expose it under both namespaces.
func NewActivationsCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "activations",
		Aliases: []string{"activation"},
		Short:   "Manage activations (end-user → environment links). Alias for the activation-lifecycle commands under `versori projects users`.",
		Long: `Manage activations on a project environment.

An activation links a previously-created end-user to a specific environment, supplying one
connection per environment system plus an optional bag of dynamic variables. This is the final
step in the per-end-user setup chain:

  1. versori users create -e <id> -n <name>                            (create end-user)
  2. versori connections create ... --external-id <id> ...             (create embedded connection)
  3. versori projects activations create ... --external-id <id> ...   (THIS COMMAND — link them)

The subcommands below are aliases for the equivalent 'versori projects users …' commands; use
whichever namespace feels more natural for the task at hand.`,
	}

	create := users.NewActivate(c)
	create.Use = "create --project <project-id> --environment <environment-name> --external-id <user-external-id> --connection <system-id>=<connection-id> [--variable key=value]... [--variables-file <path>]"
	create.Aliases = append(create.Aliases, "activate", "new")
	create.Short = "Create an activation (link an end-user to an environment with its connections + variables)"

	delete := users.NewDeactivate(c)
	delete.Use = "delete --project <project-id> --environment <environment-name> --external-id <user-external-id>"
	delete.Aliases = append(delete.Aliases, "deactivate", "rm")
	delete.Short = "Delete an activation (unlinks an end-user from an environment)"

	list := users.NewUsersList(c)
	list.Use = "list --project <project-id> --environment <environment-name>"
	list.Aliases = append(list.Aliases, "ls")
	list.Short = "List activations on a project environment"

	details := users.NewDetailsActivation(c)
	details.Use = "details --project <project-id> --environment <environment-name> --external-id <user-external-id>"
	details.Aliases = append(details.Aliases, "get", "show")
	details.Short = "Show one activation's details (connections + variables)"

	setVar := users.NewSetVariable(c)
	setVar.Use = "set-variable --project <project-id> --environment <environment-name> --external-id <user-external-id> --name <variable-name> --value <value>"
	setVar.Short = "Set a single dynamic variable on an activation"

	cmd.AddCommand(create)
	cmd.AddCommand(delete)
	cmd.AddCommand(list)
	cmd.AddCommand(details)
	cmd.AddCommand(setVar)

	return cmd
}
