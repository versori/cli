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

package users

import (
	"net/http"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

// printableActivation controls the output columns and order
// We want: displayName (first), externalId (second), activationId, userId
type printableActivation struct {
	DisplayName      string         `json:"displayName"`
	ExternalId       string         `json:"externalId"`
	ActivationId     string         `json:"activationId"`
	UserId           string         `json:"userId"`
	CreatedAt        string         `json:"createdAt"`
	DynamicVariables map[string]any `json:"dynamicVariables"`
}

func init() {
	utils.RegisterResource(printableActivation{}, []string{"DisplayName", "ExternalId", "ActivationId", "UserId", "CreatedAt"})
}

type listActivations struct {
	configFactory   *config.ConfigFactory
	projectId       string
	environmentName string
}

func NewUsersList(c *config.ConfigFactory) *cobra.Command {
	l := &listActivations{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list --project <project-id> --environment <environment-name>",
		Short: "List users (activations) linked to a project environment",
		Run:   l.Run,
	}

	flags := cmd.Flags()
	flags.StringVar(&l.projectId, "project", "", "The project ID to list users for")
	flags.StringVar(&l.environmentName, "environment", "", "The environment name within the project")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("environment")

	return cmd
}

func (l *listActivations) Run(cmd *cobra.Command, args []string) {
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

	// Call activations endpoint for the environment
	var resp []v1.Activation
	err = l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/environments/" + envId + "/activations").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list activations").WithReason(err).Done()
	}

	// Map to printable structure with desired column order
	out := make([]printableActivation, 0, len(resp))
	for _, a := range resp {
		out = append(out, printableActivation{
			DisplayName:      a.User.DisplayName,
			ExternalId:       a.User.ExternalID,
			ActivationId:     a.ID.String(),
			UserId:           a.User.ID.String(),
			CreatedAt:        a.User.CreatedAt.String(),
			DynamicVariables: a.DynamicVariables,
		})
	}

	l.configFactory.Print(out)
}
