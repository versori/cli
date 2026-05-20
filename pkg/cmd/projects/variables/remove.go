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

package variables

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type remove struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	name          string
	yes           bool
}

func NewRemove(c *config.ConfigFactory) *cobra.Command {
	r := &remove{configFactory: c}

	cmd := &cobra.Command{
		Use:     "remove --project <project-id> --name <key> [--yes]",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a single dynamic variable from the project's DynamicVariablesSchema",
		Long: `Remove a dynamic-variable declaration (and its entry in the required[] list) from the
project's DynamicVariablesSchema. Activations that previously set this key keep the value on
their record but workflow code that reads it via ctx.activation.getVariable() will continue to
return the stored value. Confirms before deleting unless --yes is passed.`,
		Run: r.Run,
	}

	f := cmd.Flags()
	r.projectId.SetFlag(f)
	f.StringVarP(&r.name, "name", "n", "", "Variable name to remove")
	f.BoolVarP(&r.yes, "yes", "y", false, "Skip the confirmation prompt")

	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func (r *remove) Run(_ *cobra.Command, _ []string) {
	projectId := r.projectId.GetFlagOrDie(".")

	schema := fetchSchema(r.configFactory, projectId)

	if existing, exists := schema.Properties[r.name]; !exists || existing == nil {
		utils.NewExitError().WithMessage(fmt.Sprintf("variable %q is not declared on project %s", r.name, projectId)).Done()
	}

	if !r.yes {
		confirmed := false
		err := elements.
			NewConfirm(fmt.Sprintf("Remove variable %q from project %s?", r.name, projectId)).
			Confirm(&confirmed)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read confirmation").WithReason(err).Done()
		}

		if !confirmed {
			fmt.Println("Aborted; no changes were made.")

			return
		}
	}

	delete(schema.Properties, r.name)
	schema.setRequired(r.name, false)

	putSchema(r.configFactory, projectId, schema)

	fmt.Printf("Removed variable %q from project %s.\n", r.name, projectId)
}
