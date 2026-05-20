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
	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
)

type list struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{configFactory: c}

	cmd := &cobra.Command{
		Use:     "list --project <project-id>",
		Aliases: []string{"ls"},
		Short:   "List the project's declared dynamic variables (name / type / required / description)",
		Long: `List the dynamic-variable keys declared on this project's DynamicVariablesSchema in
a friendly table. Activation variables set with 'versori projects users set-variable[s]' must
appear in this list — unknown keys are rejected by the platform at activation time. Use 'versori
projects variables add' to declare a new variable, or 'versori projects variables get' to dump
the raw JSON schema (escape hatch for advanced shapes like enum/default/nested object).`,
		Run: l.Run,
	}

	l.projectId.SetFlag(cmd.Flags())

	return cmd
}

func (l *list) Run(_ *cobra.Command, _ []string) {
	projectId := l.projectId.GetFlagOrDie(".")

	schema := fetchSchema(l.configFactory, projectId)

	l.configFactory.Print(schema.toRows())
}
