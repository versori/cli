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

// printableConnection is a minimal representation for table/yaml/json output.
type printableConnection struct {
	Id                   string `json:"id" yaml:"id"`
	Name                 string `json:"name" yaml:"name"`
	SystemId             string `json:"systemId" yaml:"systemId"`
	ConnectionTemplateId string `json:"connectionTemplateId" yaml:"connectionTemplateId"`
	BaseURL              string `json:"baseUrl" yaml:"baseUrl"`
}

func init() {
	utils.RegisterResource(printableConnection{}, []string{"Id", "Name", "SystemId", "ConnectionTemplateId", "BaseURL"})
}

type listConnections struct {
	configFactory   *config.ConfigFactory
	projectId       flags.ProjectId
	environmentName string
}

func NewListConnections(c *config.ConfigFactory) *cobra.Command {
	l := &listConnections{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list-connections --project <project-id> --environment <environment-name>",
		Short: "List static connections for a project environment",
		Run:   l.Run,
	}

	flags := cmd.Flags()
	l.projectId.SetFlag(flags)
	flags.StringVar(&l.environmentName, "environment", "", "The environment name within the project")

	_ = cmd.MarkFlagRequired("environment")

	return cmd
}

func (l *listConnections) Run(cmd *cobra.Command, args []string) {
	l.configFactory.LoadConfigAndContext()

	projectId := l.projectId.GetFlagOrDie(".")

	// Fetch project to resolve environment ID from name
	project := v1.Project{}
	err := l.configFactory.
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
		if e.Name == l.environmentName {
			envId = e.ID.String()

			break
		}
	}
	if envId == "" {
		utils.NewExitError().WithMessage("environment [" + l.environmentName + "] not found in project").Done()
	}

	resp := v1.ConnectionPage{}
	err = l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/projects/"+projectId+"/connections").
		WithQueryParam("env_id", envId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list connections").WithReason(err).Done()
	}

	items := make([]printableConnection, len(resp.Items))
	for i, c := range resp.Items {
		items[i] = printableConnection{
			Id:                   c.ID.String(),
			Name:                 c.Name,
			SystemId:             c.SystemID.String(),
			ConnectionTemplateId: c.EnvironmentSystemID.String(),
			BaseURL:              c.BaseURL,
		}
	}

	l.configFactory.Print(items)
}

