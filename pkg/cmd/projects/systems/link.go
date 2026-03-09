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
	"github.com/versori/cli/pkg/utils"
)

// ConnectionTemplate is a minimal representation for printing results
// Fields may be a subset of the server response; zero values will be printed if absent.
type ConnectionTemplate struct {
	Id            string `json:"id" yaml:"id"`
	Name          string `json:"name" yaml:"name"`
	ProjectId     string `json:"projectId" yaml:"projectId"`
	SystemId      string `json:"systemId" yaml:"systemId"`
	EnvironmentId string `json:"environmentId" yaml:"environmentId"`
	Dynamic       bool   `json:"dynamic" yaml:"dynamic"`
}

func init() {
	// Register headers so table output works
	utils.RegisterResource(ConnectionTemplate{}, []string{"Id", "Name", "SystemId", "EnvironmentId", "Dynamic"})
}

type linkSystem struct {
	configFactory   *config.ConfigFactory
	systemId        string
	name            string
	environmentName string
	projectId       string
	dynamic         bool
}

type linkSystemRequest struct {
	SystemId      string `json:"systemId"`
	Name          string `json:"name"`
	EnvironmentId string `json:"environmentId"`
	Dynamic       bool   `json:"dynamic"`
}

func NewSystemsAdd(c *config.ConfigFactory) *cobra.Command {
	l := &linkSystem{configFactory: c}

	cmd := &cobra.Command{
		Use:   "add --project <project-id> --system <system-id> --name <name> --environment <environment-name> [--dynamic]",
		Short: "Link a system to a project (create a connection template)",
		Run:   l.Run,
	}

	flags := cmd.Flags()
	flags.StringVar(&l.projectId, "project", "", "The project ID to link the system to")
	flags.StringVar(&l.systemId, "system", "", "The system ID to link to the project")
	flags.StringVar(&l.name, "name", "", "A name for the connection template")
	flags.StringVar(&l.environmentName, "environment", "", "The environment name within the project")
	flags.BoolVar(&l.dynamic, "dynamic", false, "Whether the connection template is dynamic")

	_ = cmd.MarkFlagRequired("system")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

func (l *linkSystem) Run(cmd *cobra.Command, args []string) {
	// Fetch project to resolve environment ID from name
	project := v1.Project{}
	err := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&project).
		WithPath("o/:organisation/projects/" + l.projectId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get project").WithReason(err).Done()
	}

	// Find environment by name
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

	payload := linkSystemRequest{
		SystemId:      l.systemId,
		Name:          l.name,
		EnvironmentId: envId,
		Dynamic:       l.dynamic,
	}

	resp := ConnectionTemplate{}
	err = l.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		WithPath("o/:organisation/projects/" + l.projectId + "/connection-templates").
		Into(&resp).
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to link system to project").WithReason(err).Done()
	}

	l.configFactory.Print(resp)
}
