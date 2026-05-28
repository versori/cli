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

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type printableProjectNotification struct {
	Name          string `json:"name"`
	Id            string `json:"id"`
	ChannelName   string `json:"channelName"`
	ChannelId     string `json:"channelId"`
	EnvironmentId string `json:"environmentId"`
}

func init() {
	utils.RegisterResource(
		printableProjectNotification{},
		[]string{"Name", "Id", "ChannelName", "ChannelId", "EnvironmentId"},
	)
}

type list struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	envName       string
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list [--project <project-id>] [--environment <name>]",
		Short: "List notifications bound to a project (optionally filtered by environment)",
		Run:   l.Run,
	}

	l.projectId.SetFlag(cmd.Flags())
	cmd.Flags().StringVar(&l.envName, "environment", "", "Filter by environment name (e.g. production, staging)")

	return cmd
}

func (l *list) Run(cmd *cobra.Command, _ []string) {
	projectId := l.projectId.GetFlagOrDie(".")

	var envId string
	if l.envName != "" {
		envId = l.resolveEnvironmentID(projectId)
	}

	resp := v1.ProjectNotificationList{}
	req := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("projects/" + projectId + "/notifications")

	if envId != "" {
		req = req.WithQueryParam("env_id", envId)
	}

	if err := req.Do(); err != nil {
		utils.NewExitError().WithMessage("failed to list project notifications").WithReason(err).Done()
	}

	items := make([]printableProjectNotification, 0, len(resp.Items))
	for _, p := range resp.Items {
		items = append(items, printableProjectNotification{
			Name:          p.Name,
			Id:            p.Id.String(),
			ChannelName:   p.ChannelName,
			ChannelId:     p.ChannelId.String(),
			EnvironmentId: p.EnvironmentId.String(),
		})
	}

	l.configFactory.Print(items)
}

// resolveEnvironmentID looks up the project and returns the ULID of the environment whose name
// matches l.envName. Exits with an error if not found.
func (l *list) resolveEnvironmentID(projectId string) string {
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

	for _, e := range project.Environments {
		if e.Name == l.envName {
			return e.ID.String()
		}
	}

	utils.NewExitError().WithMessage(fmt.Sprintf("environment %q not found on project %s", l.envName, projectId)).Done()

	return ""
}
