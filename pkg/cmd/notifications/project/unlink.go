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

package project

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type unlink struct {
	configFactory  *config.ConfigFactory
	projectId      flags.ProjectId
	notificationId string
	envName        string
	yes            bool
}

func NewUnlink(c *config.ConfigFactory) *cobra.Command {
	u := &unlink{configFactory: c}

	cmd := &cobra.Command{
		Use:     "unlink --notification-id <id> --environment <name> [--project <project-id>] [--yes]",
		Aliases: []string{"rm", "delete"},
		Short:   "Unlink a notification channel from this project + environment",
		Long: `Remove a project-notification binding so issues raised in the environment no longer alert
through the linked channel. The channel itself is not deleted; remove it separately with
'versori notifications channels delete'.

If --notification-id is omitted, the CLI lists existing bindings for the environment and prompts
you to pick one. If --environment is omitted, the CLI prompts you to pick an environment.
Confirms before deleting unless --yes is passed.`,
		Run: u.Run,
	}

	u.projectId.SetFlag(cmd.Flags())
	cmd.Flags().StringVar(&u.notificationId, "notification-id", "", "ULID of the project-notification binding to remove")
	cmd.Flags().StringVar(&u.envName, "environment", "", "Name of the project environment the binding belongs to (e.g. production, staging)")
	cmd.Flags().BoolVarP(&u.yes, "yes", "y", false, "Skip the confirmation prompt")

	return cmd
}

func (u *unlink) Run(cmd *cobra.Command, _ []string) {
	projectId := u.projectId.GetFlagOrDie(".")

	envId, envName := u.resolveEnvironment(projectId)
	bindingName := u.resolveBinding(projectId, envId)

	if !u.yes {
		confirmed := false
		err := elements.
			NewConfirm(fmt.Sprintf("Unlink binding %q (env %q) from project %s?", bindingName, envName, projectId)).
			Confirm(&confirmed)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read confirmation").WithReason(err).Done()
		}

		if !confirmed {
			fmt.Println("Aborted; no changes were made.")

			return
		}
	}

	err := u.configFactory.
		NewRequest().
		WithMethod(http.MethodDelete).
		WithPath("projects/" + projectId + "/notifications/" + u.notificationId).
		WithQueryParam("env_id", envId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to unlink notification").WithReason(err).Done()
	}

	fmt.Printf("Unlinked binding %q from environment %q on project %s.\n", bindingName, envName, projectId)
}

// resolveEnvironment returns (envID, envName). If --environment was supplied it's resolved
// against the project's environment list; otherwise the user picks interactively (auto-selecting
// when the project has exactly one environment).
func (u *unlink) resolveEnvironment(projectId string) (string, string) {
	envs := u.fetchEnvironments(projectId)

	if u.envName != "" {
		for _, e := range envs {
			if e.Name == u.envName {
				return e.ID.String(), e.Name
			}
		}

		utils.NewExitError().WithMessage(fmt.Sprintf("environment %q not found on project %s", u.envName, projectId)).Done()
	}

	if len(envs) == 0 {
		utils.NewExitError().WithMessage("project has no environments").Done()
	}

	if len(envs) == 1 {
		return envs[0].ID.String(), envs[0].Name
	}

	sort.SliceStable(envs, func(i, j int) bool {
		return envs[i].Name < envs[j].Name
	})

	sel := elements.NewListSelect("Select an environment:")
	for _, e := range envs {
		sel.AddOption(e.Name, e.ID.String())
	}

	var envId string
	if err := sel.Select(&envId); err != nil {
		utils.NewExitError().WithMessage("failed to read environment selection").WithReason(err).Done()
	}

	for _, e := range envs {
		if e.ID.String() == envId {
			return envId, e.Name
		}
	}

	return envId, envId
}

func (u *unlink) resolveBinding(projectId, envId string) string {
	bindings := u.fetchBindings(projectId, envId)

	if u.notificationId != "" {
		for _, b := range bindings {
			if b.Id.String() == u.notificationId {
				return b.Name
			}
		}

		utils.NewExitError().WithMessage(fmt.Sprintf("notification %s not found on project %s (env %s)", u.notificationId, projectId, envId)).Done()
	}

	if len(bindings) == 0 {
		utils.NewExitError().WithMessage("no bindings exist on this environment").Done()
	}

	sel := elements.NewListSelect("Select a binding to remove:")
	for _, b := range bindings {
		label := fmt.Sprintf("%s  (channel: %s)", b.Name, b.ChannelName)
		sel.AddOption(label, b.Id.String())
	}

	if err := sel.Select(&u.notificationId); err != nil {
		utils.NewExitError().WithMessage("failed to read binding selection").WithReason(err).Done()
	}

	for _, b := range bindings {
		if b.Id.String() == u.notificationId {
			return b.Name
		}
	}

	return u.notificationId
}

func (u *unlink) fetchEnvironments(projectId string) []v1.ProjectEnvironment {
	project := v1.Project{}
	err := u.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&project).
		WithPath("o/:organisation/projects/" + projectId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get project").WithReason(err).Done()
	}

	return project.Environments
}

func (u *unlink) fetchBindings(projectId, envId string) []v1.ProjectNotification {
	resp := v1.ProjectNotificationList{}
	err := u.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("projects/" + projectId + "/notifications").
		WithQueryParam("env_id", envId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list project notifications").WithReason(err).Done()
	}

	return resp.Items
}
