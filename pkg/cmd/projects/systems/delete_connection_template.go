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

package systems

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/versori/cli/pkg/cmd/flags"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type deleteConnectionTemplate struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	templateId    string
}

func NewDeleteConnectionTemplate(c *config.ConfigFactory) *cobra.Command {
	d := &deleteConnectionTemplate{configFactory: c}

	cmd := &cobra.Command{
		Use:   "delete-connection-template --project <project-id> --template <template-id>",
		Short: "Delete a connection template from a project",
		Run:   d.Run,
	}

	flags := cmd.Flags()
	d.projectId.SetFlag(flags)
	flags.StringVar(&d.templateId, "template", "", "The connection template ID to delete")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("template")

	return cmd
}

func (d *deleteConnectionTemplate) Run(cmd *cobra.Command, args []string) {
	projectId := d.projectId.GetFlagOrDie(".")

	err := d.configFactory.
		NewRequest().
		WithMethod(http.MethodDelete).
		WithPath("o/:organisation/projects/" + projectId + "/connection-templates/" + d.templateId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to delete connection template").WithReason(err).Done()
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "deleted")
}
