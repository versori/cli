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
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/utils"
)

type ProjectDetails struct {
	Id           string              `json:"id"`
	Name         string              `json:"name"`
	Deployed     bool                `json:"deployed"`
	Starred      bool                `json:"starred"`
	Environments []EnvironmentDetail `json:"environments"`
}

type EnvironmentDetail struct {
	Id              string               `json:"id"`
	Name            string               `json:"name"`
	PublicUrl       string               `json:"publicUrl"`
	Status          string               `json:"status"`
	Provisioner     string               `json:"provisioner"`
	Config          v1.EnvironmentConfig `json:"config"`
	DeployedVersion *v1.ProjectVersion   `json:"deployedVersion,omitempty"`
}

func init() {
	utils.RegisterResource(ProjectDetails{}, []string{"Name", "Id", "Deployed", "Starred", "Environments.Name"})
}

type details struct {
	configFactory *config.ConfigFactory
}

func NewDetails(c *config.ConfigFactory) *cobra.Command {
	d := &details{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "details <project-id>",
		Short: "Get details for a project. Pass in - to read the project id from stdin",
		Run:   d.Run,
	}

	return cmd
}

func (d *details) Run(cmd *cobra.Command, args []string) {
	project := v1.Project{}
	var projectId string

	if len(args) < 1 {
		projectList := ProjectListResponse{}

		err := d.configFactory.
			NewRequest().
			WithMethod(http.MethodGet).
			Into(&projectList).
			WithPath("o/:organisation/projects").
			Do()
		if err != nil {
			utils.NewExitError().WithMessage("failed to list projects").WithReason(err).Done()
		}

		projectSelector := elements.NewListSelect("Select a project:")
		for _, p := range projectList.Projects {
			projectSelector.AddOption(p.Name, p.ID.String())
		}

		err = projectSelector.Select(&projectId)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read project selection").WithReason(err).Done()
		}

	} else {
		projectId = args[0]
		if projectId == "-" {
			b, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				utils.NewExitError().WithMessage("failed to read project id from stdin").WithReason(err).Done()
			}

			projectId = strings.TrimSpace(string(b))
		}

	}

	err := d.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&project).
		WithPath("o/:organisation/projects/" + projectId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get project").WithReason(err).Done()
	}

	pd := ProjectDetails{
		Id:           project.ID.String(),
		Name:         project.Name,
		Deployed:     deployed(project.Environments),
		Starred:      project.Starred,
		Environments: envDetails(project.Environments),
	}

	d.configFactory.Print(pd)
}

func envDetails(envs []v1.ProjectEnvironment) []EnvironmentDetail {
	details := make([]EnvironmentDetail, 0, len(envs))
	for _, e := range envs {
		details = append(details, EnvironmentDetail{
			Id:              e.ID.String(),
			Name:            e.Name,
			PublicUrl:       e.PublicUrl,
			Provisioner:     e.ExecutionPool,
			Status:          e.Status,
			Config:          e.Config,
			DeployedVersion: e.DeployedVersion,
		})
	}

	return details
}
