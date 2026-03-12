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

package projects

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/versori/cli/pkg/cmd/flags"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/projects/assets"
	"github.com/versori/cli/pkg/utils"
)

type Deploy struct {
	configFactory *config.ConfigFactory
	directory     string
	env           string
	versionName   string
	description   string
	dryRun        bool
	uploadAssets  bool
	projectId     flags.ProjectId
}

func NewDeploy(c *config.ConfigFactory) *cobra.Command {
	d := &Deploy{configFactory: c}

	cmd := &cobra.Command{
		Use:   "deploy [--project <project-id>] --environment <env> [--directory <directory>]",
		Short: "Deploy the project to versori",
		Run:   d.Run,
	}

	flags := cmd.Flags()
	d.projectId.SetFlag(flags)
	flags.StringVarP(&d.directory, "directory", "d", ".", "The directory containing the project files")
	flags.StringVar(&d.env, "environment", "", "The project environment to deploy to (e.g. production, staging)")
	flags.StringVar(&d.versionName, "version", "", "Name of the version to create (default: current time in UTC, e.g. 2006-01-02-15-04-05)")
	flags.StringVar(&d.description, "description", "", "Description of the new version.")
	flags.BoolVar(&d.dryRun, "dry-run", false, "Print files that would be uploaded without actually deploying")
	flags.BoolVar(&d.uploadAssets, "assets", false, "Also upload assets from the "+assets.DefaultAssetsDir+" directory")

	_ = cmd.MarkFlagRequired("environment")

	return cmd
}

func (d *Deploy) Run(cmd *cobra.Command, args []string) {
	fullPath := d.directory
	if !filepath.IsAbs(d.directory) {
		currentDir, err := os.Getwd()
		if err != nil {
			utils.NewExitError().WithMessage("failed to get current directory").WithReason(err).Done()
		}

		fullPath = filepath.Join(currentDir, d.directory)
	}

	projectId := d.projectId.GetFlagOrDie(fullPath)

	if d.versionName == "" {
		d.versionName = time.Now().UTC().Format("2006-01-02-15-04-05")
	}

	files, err := utils.CollectFiles(fullPath, d.dryRun)
	if err != nil {
		utils.NewExitError().WithMessage("failed to collect local files").WithReason(err).Done()
	}

	// If dry-run, print files and exit
	if d.dryRun {
		fmt.Println("Files that would be uploaded:")
		for _, file := range files {
			fmt.Println("  " + file.Filename)
		}

		return
	}

	payload := v1.DeployProjectJSONRequestBody{
		Files:       files,
		Labels:      map[string]string{},
		VersionName: d.versionName,
	}

	if d.description != "" {
		payload.VersionDescription = &d.description
	}

	requestPath := "o/:organisation/projects/" + projectId + "/deploy"

	err = d.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		WithPath(requestPath).
		WithQueryParam("project_env", d.env).
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to deploy").WithReason(err).Done()
	}

	if d.uploadAssets {
		d.syncAssets(projectId, fullPath)
	}
}

func (d *Deploy) syncAssets(projectId, fullPath string) {
	orgId := d.configFactory.Context.OrganisationId
	assetDir := filepath.Join(fullPath, assets.DefaultAssetsDir)

	assetFiles, err := assets.CollectAssetFiles(assetDir)
	if err != nil {
		utils.NewExitError().WithMessage("failed to collect asset files").WithReason(err).Done()
	}

	if len(assetFiles) == 0 {
		fmt.Println("No asset files found in " + assets.DefaultAssetsDir)
		return
	}

	if d.dryRun {
		fmt.Println("Assets that would be uploaded:")
		for _, f := range assetFiles {
			fmt.Println("  " + filepath.Base(f))
		}
		return
	}

	for _, f := range assetFiles {
		fmt.Printf("Uploading asset %q...\n", filepath.Base(f))
		if uploadErr := assets.UploadAssetFile(d.configFactory, orgId, projectId, f, "research/documents"); uploadErr != nil {
			utils.NewExitError().WithMessage("failed to upload asset").WithReason(uploadErr).Done()
		}
		fmt.Printf("Successfully uploaded asset %q\n", filepath.Base(f))
	}
}
