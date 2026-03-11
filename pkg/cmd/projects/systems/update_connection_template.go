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
	"net/http"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type updateConnectionTemplate struct {
	configFactory       *config.ConfigFactory
	projectId           flags.ProjectId
	templateId          string
	name                string
	dynamic             bool
	authSchemeConfigIds string
}

func NewUpdateConnectionTemplate(c *config.ConfigFactory) *cobra.Command {
	u := &updateConnectionTemplate{configFactory: c}

	cmd := &cobra.Command{
		Use:   "update-connection-template --project <project-id> --template <template-id> [--name <name>] [--dynamic] [--auth-scheme-config-id <name>]",
		Short: "Update a connection template in a project",
		Run:   u.Run,
	}

	flags := cmd.Flags()
	u.projectId.SetFlag(flags)
	flags.StringVar(&u.templateId, "template", "", "The connection template ID to update")

	flags.StringVar(&u.name, "name", "", "New name for the connection template")
	flags.BoolVar(&u.dynamic, "dynamic", false, "Whether the connection template is dynamic")
	flags.StringVar(&u.authSchemeConfigIds, "auth-scheme-config-id", "", "Auth scheme config ID to associate with the connection template")

	_ = cmd.MarkFlagRequired("template")

	return cmd
}

func (u *updateConnectionTemplate) Run(cmd *cobra.Command, args []string) {
	u.configFactory.LoadConfigAndContext()

	projectId := u.projectId.GetFlagOrDie(".")

	payload := v1.UpdateConnectionTemplate{}
	updatesSet := false

	if u.name != "" {
		payload.Name = &u.name
		updatesSet = true
	}

	if u.authSchemeConfigIds != "" {
		payload.AuthSchemeConfigIds = []string{u.authSchemeConfigIds}
		updatesSet = true
	}

	if cmd.Flags().Changed("dynamic") {
		payload.Dynamic = &u.dynamic
		updatesSet = true
	}

	if !updatesSet {
		utils.NewExitError().WithMessage("Nothing to update, please set at least one update flag").Done()
	}

	err := u.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		WithPath("o/:organisation/projects/" + projectId + "/connection-templates/" + u.templateId).
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to update connection template").WithReason(err).Done()
	}
}
