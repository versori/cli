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

package assets

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/versori/cli/pkg/cmd/flags"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type printableAsset struct {
	Name         string
	Size         string
	ContentType  string
	Path         string
	DownloadURL  string
	LastModified string
}

func init() {
	utils.RegisterResource(printableAsset{}, []string{"Name", "Size", "ContentType", "Path"})
}

type list struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List assets for a project",
		Long:  "List retrieves all assets for a project from the Versori platform.",
		Run:   l.Run,
	}

	flags := cmd.Flags()
	l.projectId.SetFlag(flags)

	return cmd
}

func (l *list) Run(cmd *cobra.Command, args []string) {
	orgId := l.configFactory.Context.OrganisationId
	projectId := l.projectId.GetFlagOrDie(".")

	resp, err := ListAssets(l.configFactory, orgId, projectId)
	if err != nil {
		utils.NewExitError().WithMessage("failed to list assets").WithReason(err).Done()
	}

	items := make([]printableAsset, 0, len(resp.Assets))
	for _, a := range resp.Assets {
		items = append(items, printableAsset{
			Name:         a.Name,
			Size:         fmt.Sprintf("%d", a.Size),
			ContentType:  a.ContentType,
			Path:         a.Path,
			DownloadURL:  a.DownloadURL,
			LastModified: a.LastModified,
		})
	}

	l.configFactory.Print(items)
}
