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
	"net/http"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type List struct {
	configFactory *config.ConfigFactory
}

type ProjectListResponse struct {
	Projects []v1.Project `json:"projects"`
}

type ProjectSummary struct {
	Id           string   `json:"id"`
	Name         string   `json:"name"`
	Deployed     bool     `json:"deployed"`
	Environments []string `json:"environments"`
}

func init() {
	utils.RegisterResource(ProjectSummary{}, []string{"Name", "Id", "Deployed", "Environments"})
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &List{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all projects in the current context",
		Run:   l.Run,
	}

	return cmd
}

func (a *List) Run(cmd *cobra.Command, args []string) {
	projectList := ProjectListResponse{}

	err := a.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&projectList).
		WithPath("o/:organisation/projects").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list projects").WithReason(err).Done()
	}

	printableProjects := make([]ProjectSummary, 0, len(projectList.Projects))
	for _, p := range projectList.Projects {
		printableProjects = append(printableProjects, ProjectSummary{
			Id:           p.ID.String(),
			Name:         p.Name,
			Deployed:     deployed(p.Environments),
			Environments: envNames(p.Environments),
		})
	}

	slices.SortFunc(printableProjects, func(a, b ProjectSummary) int {
		return strings.Compare(a.Name, b.Name)
	})

	a.configFactory.Print(printableProjects)
}

func envNames(envs []v1.ProjectEnvironment) []string {
	names := make([]string, 0, len(envs))
	for _, e := range envs {
		names = append(names, e.Name)
	}

	return names
}

func deployed(envs []v1.ProjectEnvironment) bool {
	for _, e := range envs {
		if e.Status == "running" {
			return true
		}
	}

	return false
}
