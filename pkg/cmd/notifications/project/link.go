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
	"github.com/versori/cli/pkg/ulid"
	"github.com/versori/cli/pkg/utils"
)

type link struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	channelId     string
	envName       string
	name          string
}

func NewLink(c *config.ConfigFactory) *cobra.Command {
	l := &link{configFactory: c}

	cmd := &cobra.Command{
		Use:   "link --channel-id <id> --environment <name> [--name <label>] [--project <project-id>]",
		Short: "Link an existing notification channel to this project + environment",
		Long: `Link an existing notification channel (created with 'versori notifications channels create')
to a project + environment. After linking, issues created in that environment by workflow code
('ctx.createIssue()' or '.catch()' blocks) trigger an email through the channel.

If --channel-id or --environment is omitted, the CLI presents an interactive picker of the
available channels/environments by name. --project defaults from .versori when inside a synced
project directory.`,
		Run: l.Run,
	}

	l.projectId.SetFlag(cmd.Flags())
	cmd.Flags().StringVar(&l.channelId, "channel-id", "", "ULID of the notification channel to link (prompts a picker if omitted)")
	cmd.Flags().StringVar(&l.envName, "environment", "", "Name of the project environment (e.g. production, staging; prompts a picker if omitted)")
	cmd.Flags().StringVar(&l.name, "name", "", "Display name for this link (defaults to the channel name)")

	return cmd
}

func (l *link) Run(cmd *cobra.Command, _ []string) {
	projectId := l.projectId.GetFlagOrDie(".")

	channelName := l.resolveChannel()
	envId, envName := l.resolveEnvironment(projectId)

	if l.name == "" {
		l.name = channelName
	}

	channelULID, err := ulid.Parse(l.channelId)
	if err != nil {
		utils.NewExitError().WithMessage("invalid --channel-id").WithReason(err).Done()
	}

	envULID, err := ulid.Parse(envId)
	if err != nil {
		utils.NewExitError().WithMessage("failed to parse resolved environment ID").WithReason(err).Done()
	}

	payload := v1.CreateProjectNotificationJSONRequestBody{
		ChannelId:     channelULID,
		EnvironmentId: envULID,
		Name:          l.name,
	}

	resp := v1.ProjectNotification{}
	err = l.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		Into(&resp).
		WithPath("projects/" + projectId + "/notifications").
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to link channel to project").WithReason(err).Done()
	}

	fmt.Printf("Linked channel %q to environment %q on project %s (notification id: %s, name: %q).\n",
		channelName, envName, projectId, resp.Id.String(), resp.Name)
}

// resolveChannel returns the channel name (for the success message) and ensures l.channelId is set.
// If l.channelId is already provided via flag, the channel name is fetched for display.
// Otherwise, the user picks one interactively.
func (l *link) resolveChannel() string {
	channels := l.fetchChannels()

	if l.channelId != "" {
		for _, ch := range channels {
			if ch.Id.String() == l.channelId {
				return ch.Name
			}
		}

		utils.NewExitError().WithMessage(fmt.Sprintf("channel %s not found in current organisation", l.channelId)).Done()
	}

	if len(channels) == 0 {
		utils.NewExitError().WithMessage("no notification channels exist; create one first with 'versori notifications channels create'").Done()
	}

	sel := elements.NewListSelect("Select a notification channel:")
	for _, ch := range channels {
		to := ""
		if ch.Config.Email != nil {
			to = ch.Config.Email.To
		}

		label := fmt.Sprintf("%s  (%s)", ch.Name, to)
		sel.AddOption(label, ch.Id.String())
	}

	if err := sel.Select(&l.channelId); err != nil {
		utils.NewExitError().WithMessage("failed to read channel selection").WithReason(err).Done()
	}

	for _, ch := range channels {
		if ch.Id.String() == l.channelId {
			return ch.Name
		}
	}

	return l.channelId
}

// resolveEnvironment returns (envID, envName). If --environment was supplied it's resolved
// against the project's environment list; otherwise the user picks interactively (auto-selecting
// when the project has exactly one environment).
func (l *link) resolveEnvironment(projectId string) (string, string) {
	envs := l.fetchEnvironments(projectId)

	if l.envName != "" {
		for _, e := range envs {
			if e.Name == l.envName {
				return e.ID.String(), e.Name
			}
		}

		utils.NewExitError().WithMessage(fmt.Sprintf("environment %q not found on project %s", l.envName, projectId)).Done()
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

func (l *link) fetchChannels() []v1.NotificationChannel {
	resp := v1.NotificationChannelList{}
	err := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/notification_channels").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list notification channels").WithReason(err).Done()
	}

	return resp.Items
}

func (l *link) fetchEnvironments(projectId string) []v1.ProjectEnvironment {
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

	return project.Environments
}
