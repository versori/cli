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
	"github.com/versori/cli/pkg/ulid"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

func init() {
	utils.RegisterResource(bigActivation{}, []string{"ID", "User.ID", "User.DisplayName", "User.ExternalID", "DynamicVariables"})
}

type bigActivation struct {
	Connections      []v1.Connection     `json:"connections"`
	DynamicVariables v1.DynamicVariables `json:"dynamicVariables"`
	ID               ulid.ULID           `json:"id"`
	User             v1.EndUser          `json:"user"`
}

type detailsActivation struct {
	configFactory   *config.ConfigFactory
	projectId       string
	environmentName string
	user            string
}

func NewDetailsActivation(c *config.ConfigFactory) *cobra.Command {
	l := &detailsActivation{configFactory: c}

	cmd := &cobra.Command{
		Use:   "details --project <project-id> --environment <environment-name> --external-id <user-id>",
		Short: "Get the details for an individual end user of a project",
		Run:   l.Run,
	}

	flags := cmd.Flags()
	flags.StringVar(&l.projectId, "project", "", "The project ID to list users for")
	flags.StringVar(&l.environmentName, "environment", "", "The environment name within the project")
	flags.StringVar(&l.user, "external-id", "", "The external ID of the user")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("external-id")

	return cmd
}

func (l *detailsActivation) Run(cmd *cobra.Command, args []string) {
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

	// find activationID based on external id for the user
	activationId := ""
	for _, a := range resp {
		if a.User.ExternalID == l.user {
			activationId = a.ID.String()

			break
		}
	}

	if activationId == "" {
		utils.NewExitError().WithMessage("No user found for external ID: " + l.user).Done()
	}

	activation := v1.Activation{}
	err = l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&activation).
		WithPath("o/:organisation/environments/" + envId + "/activations/" + activationId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get activation").WithReason(err).Done()
	}

	ba := bigActivation{
		Connections:      []v1.Connection{},
		DynamicVariables: activation.DynamicVariables,
		ID:               activation.ID,
		User:             activation.User,
	}

	if activation.Connections == nil {
		l.configFactory.Print(activation)

		return
	}

	for _, c := range *activation.Connections {
		credentails := v1.Connection{}

		err = l.configFactory.
			NewRequest().
			WithMethod(http.MethodGet).
			Into(&credentails).
			WithPath("o/:organisation/connections/" + c.ID.String()).
			Do()
		if err != nil {
			utils.NewExitError().WithMessage("failed to get connection").WithReason(err).Done()
		}

		ba.Connections = append(ba.Connections, credentails)
	}

	l.configFactory.Print(ba)
}
