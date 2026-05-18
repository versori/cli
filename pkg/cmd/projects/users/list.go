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
	"sort"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

// printableActivation is the table-printable view, one row per end-user. Active rows fill in
// ActivationId + DynamicVariables; inactive rows leave those blank. Status is always populated.
type printableActivation struct {
	DisplayName      string         `json:"displayName"`
	ExternalId       string         `json:"externalId"`
	Status           string         `json:"status"`
	ActivationId     string         `json:"activationId,omitempty"`
	UserId           string         `json:"userId"`
	CreatedAt        string         `json:"createdAt"`
	DynamicVariables map[string]any `json:"dynamicVariables,omitempty"`
}

func init() {
	utils.RegisterResource(printableActivation{}, []string{"DisplayName", "ExternalId", "Status", "ActivationId", "UserId", "CreatedAt"})
}

type listActivations struct {
	configFactory   *config.ConfigFactory
	projectId       flags.ProjectId
	environmentName string
}

func NewUsersList(c *config.ConfigFactory) *cobra.Command {
	l := &listActivations{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list --project <project-id> --environment <environment-name>",
		Short: "List all end-users with their activation status (active/inactive) for a project environment",
		Long: `List every end-user in the organisation alongside their activation status on the given
project environment. Active rows include the ActivationId and the activation's dynamic variables;
inactive rows show the end-user but leave activation-specific fields blank. Use -o yaml or
-o json to see the full row including DynamicVariables for active users.`,
		Run: l.Run,
	}

	flags := cmd.Flags()
	l.projectId.SetFlag(flags)
	flags.StringVar(&l.environmentName, "environment", "", "The environment name within the project")

	_ = cmd.MarkFlagRequired("environment")

	return cmd
}

func (l *listActivations) Run(cmd *cobra.Command, args []string) {
	projectId := l.projectId.GetFlagOrDie(".")

	// Resolve environment ID by name from the project record.
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

	// Active activations on this environment.
	var activations []v1.Activation
	err = l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&activations).
		WithPath("o/:organisation/environments/" + envId + "/activations").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list activations").WithReason(err).Done()
	}

	// All org-wide end-users (so inactive ones surface alongside active activations).
	usersResp := v1.EndUserPage{}
	err = l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&usersResp).
		WithPath("o/:organisation/users").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list users").WithReason(err).Done()
	}

	// Index activations by user ULID so the merge below is O(N+M) rather than O(N*M).
	activationByUserId := make(map[string]*v1.Activation, len(activations))
	for i := range activations {
		activationByUserId[activations[i].User.ID.String()] = &activations[i]
	}

	out := make([]printableActivation, 0, len(usersResp.Users))
	for _, u := range usersResp.Users {
		row := printableActivation{
			DisplayName: u.DisplayName,
			ExternalId:  u.ExternalID,
			Status:      "inactive",
			UserId:      u.ID.String(),
			CreatedAt:   u.CreatedAt.String(),
		}
		if a, ok := activationByUserId[u.ID.String()]; ok {
			row.Status = "active"
			row.ActivationId = a.ID.String()
			row.DynamicVariables = a.DynamicVariables
		}
		out = append(out, row)
	}

	// Stable ordering: active first (so newly-activated users surface at the top), then
	// alphabetically by display name within each group.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Status != out[j].Status {
			return out[i].Status == "active"
		}
		return out[i].DisplayName < out[j].DisplayName
	})

	l.configFactory.Print(out)
}
