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

package versions

import (
	"strconv"
	"time"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type VersionSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreatedAt   string `json:"createdAt"`
}

func init() {
	utils.RegisterResource(VersionSummary{}, []string{"Name", "State", "ID", "CreatedAt"})
}

type List struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	limit         int
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &List{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists most recent versions",
		Run:   l.Run,
	}

	flags := cmd.Flags()

	l.projectId.SetFlag(flags)
	flags.IntVar(&l.limit, "limit", 20, "How many versions to list")

	return cmd
}

func (l *List) Run(_ *cobra.Command, _ []string) {
	projectId := l.projectId.GetFlagOrDie(".")
	versionPage := v1.VersionPage{}

	path := "o/:organisation/projects/" + projectId + "/versions"

	err := l.configFactory.
		NewRequest().
		WithMethod("GET").
		WithQueryParam("first", strconv.Itoa(l.limit)).
		Into(&versionPage).
		WithPath(path).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list versions").WithReason(err).Done()
	}

	versionSummaries := make([]VersionSummary, 0, len(versionPage.Items))

	for _, v := range versionPage.Items {
		versionSummaries = append(versionSummaries, VersionSummary{
			ID:          v.ID.String(),
			Name:        v.Name,
			Description: v.Description,
			State:       string(v.State),
			CreatedAt:   v.CreatedAt.Format(time.RFC3339),
		})
	}

	l.configFactory.Print(versionSummaries)
}
