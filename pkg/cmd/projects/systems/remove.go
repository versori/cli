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

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type removeSystem struct {
	configFactory *config.ConfigFactory
	projectId     string
	templateId    string
}

func NewSystemsRemove(c *config.ConfigFactory) *cobra.Command {
	r := &removeSystem{configFactory: c}

	cmd := &cobra.Command{
		Use:   "remove --project <project-id> --template <connection-template-id>",
		Short: "Remove a system (connection template) from a project",
		Run:   r.Run,
	}

	flags := cmd.Flags()
	flags.StringVar(&r.projectId, "project", "", "The project ID from which to remove the system")
	flags.StringVar(&r.templateId, "template", "", "The connection template ID to remove")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("template")

	return cmd
}

func (r *removeSystem) Run(cmd *cobra.Command, args []string) {
	// Perform DELETE request
	err := r.configFactory.
		NewRequest().
		WithMethod(http.MethodDelete).
		WithPath("o/:organisation/projects/" + r.projectId + "/connection-templates/" + r.templateId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to remove system from project").WithReason(err).Done()
	}

	// On success, print nothing or a short confirmation
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "removed")
}
