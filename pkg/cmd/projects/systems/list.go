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

// ProjectConnectionTemplatesResponse models the list response for project connection templates
type ProjectConnectionTemplatesResponse struct {
	TotalCount int                             `json:"totalCount"`
	Next       string                          `json:"next"`
	Prev       string                          `json:"prev"`
	Items      []ProjectConnectionTemplateItem `json:"items"`
}

// ProjectConnectionTemplateItem represents a single connection template (system linked to project)
// Note: intentionally omitting authSchemeConfigs for now per requirements
type ProjectConnectionTemplateItem struct {
	Id                   string                `json:"id"`
	ConnectionTemplateId string                `json:"connectionTemplateId"`
	Name                 string                `json:"name"`
	Domain               string                `json:"domain"`
	Dynamic              bool                  `json:"dynamic"`
	TemplateBaseUrl      string                `json:"templateBaseUrl"`
	AuthSchemeConfigs    []v1.AuthSchemeConfig `json:"authSchemeConfigs"`
}

// Register a concise printable view for table outputs
func init() {
	utils.RegisterResource(ProjectConnectionTemplateItem{}, []string{"Name", "Id", "ConnectionTemplateId", "TemplateBaseUrl", "Dynamic", "AuthSchemeConfigs.Type"})
}

type listSystems struct {
	configFactory   *config.ConfigFactory
	projectId       flags.ProjectId
	environmentName string
}

func NewSystemsList(c *config.ConfigFactory) *cobra.Command {
	l := &listSystems{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list --project <project-id> --environment <environment-name>",
		Short: "List systems linked to a project",
		Run:   l.Run,
	}

	flags := cmd.Flags()
	l.projectId.SetFlag(flags)
	flags.StringVar(&l.environmentName, "environment", "", "The environment name within the project")

	_ = cmd.MarkFlagRequired("environment")

	return cmd
}

func (l *listSystems) Run(cmd *cobra.Command, args []string) {
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

	// Resolve environment ID by name
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

	resp := ProjectConnectionTemplatesResponse{}
	err = l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/projects/"+projectId+"/connection-templates").
		WithQueryParam("env_id", envId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list project systems").WithReason(err).Done()
	}

	// Print only the items as requested
	l.configFactory.Print(resp.Items)
}
