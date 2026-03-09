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

type download struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	directory     string
	assetName     string
}

func NewDownload(c *config.ConfigFactory) *cobra.Command {
	d := &download{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "download --project <id> --asset <name>",
		Short: "Download an asset from the Versori platform",
		Long:  "Download retrieves an asset by name from the Versori platform and saves it to the specified directory.",
		Run:   d.Run,
	}

	f := cmd.Flags()
	d.projectId.SetFlag(f)
	f.StringVarP(&d.directory, "directory", "d", DefaultAssetsDir, "Directory to save the downloaded asset")
	f.StringVarP(&d.assetName, "asset", "a", "", "Name of the asset to download (required)")

	_ = cmd.MarkFlagRequired("asset")

	return cmd
}

func (d *download) Run(cmd *cobra.Command, args []string) {
	orgId := d.configFactory.Context.OrganisationId
	projectId := d.projectId.GetFlagOrDie(".")

	resp, err := ListAssets(d.configFactory, orgId, projectId)
	if err != nil {
		utils.NewExitError().WithMessage("failed to list assets").WithReason(err).Done()
	}

	var found *Asset
	for i := range resp.Assets {
		if resp.Assets[i].Name == d.assetName {
			found = &resp.Assets[i]
			break
		}
	}

	if found == nil {
		utils.NewExitError().WithMessage(fmt.Sprintf("asset %q not found in project", d.assetName)).Done()

		// this return is never hit because the exit error will exit the program on .Done()
		// this is only here to appease the linter
		return
	}

	if found.DownloadURL == "" {
		utils.NewExitError().WithMessage(fmt.Sprintf("asset %q has no download URL", d.assetName)).Done()
	}

	if err := DownloadAssetToFile(found.DownloadURL, found.Name, d.directory); err != nil {
		utils.NewExitError().WithMessage("failed to download asset").WithReason(err).Done()
	}

	fmt.Printf("Successfully downloaded %q to %s\n", found.Name, d.directory)
}
