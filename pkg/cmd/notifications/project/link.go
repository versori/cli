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

var validNotificationSeverities = []string{"critical", "high", "medium", "low"}

// notificationFilters mirrors the server-side NotificationFilters struct. All fields are optional;
// an empty struct (all zero values) matches every issue.
type notificationFilters struct {
	Severities       []string `json:"severities,omitempty"`
	Title            string   `json:"title,omitempty"`
	TitleMatchCase   bool     `json:"titleMatchCase,omitempty"`
	Message          string   `json:"message,omitempty"`
	MessageMatchCase bool     `json:"messageMatchCase,omitempty"`
}

// createNotificationBody is the request body for creating a project notification link. It is defined
// locally because the generated CreateProjectNotification type has no Filters field.
type createNotificationBody struct {
	ChannelId     ulid.ULID            `json:"channel_id"`
	EnvironmentId ulid.ULID            `json:"environment_id"`
	Name          string               `json:"name"`
	Filters       *notificationFilters `json:"filters,omitempty"`
}

type link struct {
	configFactory   *config.ConfigFactory
	projectId       flags.ProjectId
	channelId       string
	envName         string
	name            string
	severities      []string
	filterTitle     string
	filterTitleCase bool
	filterMessage   string
	filterMsgCase   bool
}

func NewLink(c *config.ConfigFactory) *cobra.Command {
	l := &link{configFactory: c}

	cmd := &cobra.Command{
		Use:   "link --channel-id <id> --environment <name> [--name <label>] [--project <project-id>]",
		Short: "Link an existing notification channel to this project + environment",
		Long: `Link an existing notification channel (created with 'versori notifications channels create')
to a project + environment. After linking, issues created in that environment by workflow code
('ctx.createIssue()' or '.catch()' blocks) also trigger an email through the channel. Issues
are always visible in the Issues UI without a linked channel — linking only adds email delivery.

If --channel-id or --environment is omitted, the CLI presents an interactive picker of the
available channels/environments by name. --project defaults from .versori when inside a synced
project directory.`,
		Run: l.Run,
	}

	f := cmd.Flags()
	l.projectId.SetFlag(f)
	f.StringVar(&l.channelId, "channel-id", "", "ULID of the notification channel to link (prompts a picker if omitted)")
	f.StringVar(&l.envName, "environment", "", "Name of the project environment (e.g. production, staging; prompts a picker if omitted)")
	f.StringVar(&l.name, "name", "", "Display name for this link (defaults to the channel name)")
	f.StringSliceVar(&l.severities, "severity", nil, "Only email for these severities (comma-separated or repeatable): critical, high, medium, low")
	f.StringVar(&l.filterTitle, "filter-title", "", "Only email when issue title contains this substring")
	f.BoolVar(&l.filterTitleCase, "filter-title-case", false, "Make --filter-title case-sensitive")
	f.StringVar(&l.filterMessage, "filter-message", "", "Only email when issue message contains this substring")
	f.BoolVar(&l.filterMsgCase, "filter-message-case", false, "Make --filter-message case-sensitive")

	return cmd
}

func (l *link) Run(cmd *cobra.Command, _ []string) {
	projectId := l.projectId.GetFlagOrDie(".")

	for _, s := range l.severities {
		found := false
		for _, valid := range validNotificationSeverities {
			if s == valid {
				found = true
				break
			}
		}
		if !found {
			utils.NewExitError().WithMessage(fmt.Sprintf("invalid --severity %q (must be one of %v)", s, validNotificationSeverities)).Done()
		}
	}

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

	payload := createNotificationBody{
		ChannelId:     channelULID,
		EnvironmentId: envULID,
		Name:          l.name,
		Filters:       l.buildFilters(),
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

// buildFilters constructs a notificationFilters from the filter flags. Returns nil when no filter
// flags are set so the API receives no filters field (matches all issues).
func (l *link) buildFilters() *notificationFilters {
	empty := len(l.severities) == 0 &&
		l.filterTitle == "" && !l.filterTitleCase &&
		l.filterMessage == "" && !l.filterMsgCase

	if empty {
		return nil
	}

	return &notificationFilters{
		Severities:       l.severities,
		Title:            l.filterTitle,
		TitleMatchCase:   l.filterTitleCase,
		Message:          l.filterMessage,
		MessageMatchCase: l.filterMsgCase,
	}
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
