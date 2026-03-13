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
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/ulid"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type connectSystem struct {
	configFactory        *config.ConfigFactory
	projectId            flags.ProjectId
	envName              string
	connectionTemplateId string
	connectionId         string
}

func NewSystemsConnect(c *config.ConfigFactory) *cobra.Command {
	cs := &connectSystem{configFactory: c}

	cmd := &cobra.Command{
		Use:   "connect --project <project-id> --system <system-id>",
		Short: "Connect to a system",
		Run:   cs.Run,
	}

	flags := cmd.Flags()
	cs.projectId.SetFlag(flags)
	flags.StringVar(&cs.envName, "environment", "", "The environment name within the project")
	flags.StringVar(&cs.connectionTemplateId, "template-id", "", "ID of the connection template to connect to")
	flags.StringVar(&cs.connectionId, "connection-id", "", "ID of the connection to connect to")

	_ = cmd.MarkFlagRequired("system")
	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("template-id")

	return cmd
}

func (c *connectSystem) Run(cmd *cobra.Command, args []string) {
	projectId := c.projectId.GetFlagOrDie(".")

	// Fetch project to resolve environment ID from name
	project := v1.Project{}
	err := c.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&project).
		WithPath("o/:organisation/projects/" + projectId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get project").WithReason(err).Done()
	}

	envId := ""
	for _, e := range project.Environments {
		if e.Name == c.envName {
			envId = e.ID.String()

			break
		}
	}
	if envId == "" {
		utils.NewExitError().WithMessage("environment [" + c.envName + "] not found in project").Done()
	}

	templateUlid, err := ulid.Parse(c.connectionTemplateId)
	if err != nil {
		utils.NewExitError().
			WithMessage("connection template ID must be a valid ULID. If you are unsure of the connection template ID, run `versori projects systems list --project " + projectId + " --environment " + c.envName + "").
			WithReason(err).Done()
	}

	req := v1.LinkConnectionToEnvironmentJSONRequestBody{
		EnvironmentSystemID: templateUlid,
	}

	err = c.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		WithPath("o/:organisation/connections/" + c.connectionId + "/link").
		JSONBody(req).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("Failed to link connection to environment").WithReason(err).Done()
	}
}
