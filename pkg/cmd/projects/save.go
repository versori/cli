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

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/cmd/projects/assets"
	"github.com/versori/cli/pkg/utils"
)

type Save struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	directory     string
	dryRun        bool
	uploadAssets  bool
}

type saveRequest struct {
	Files  []v1.File         `json:"files"`
	Labels map[string]string `json:"labels"`
}

func NewSave(c *config.ConfigFactory) *cobra.Command {
	s := &Save{configFactory: c}

	cmd := &cobra.Command{
		Use:   "save [--project <project-id>] [--directory <directory>]",
		Short: "Save the project files to versori (no deploy)",
		Run:   s.Run,
	}

	f := cmd.Flags()
	s.projectId.SetFlag(f)
	f.StringVarP(&s.directory, "directory", "d", ".", "The directory containing the project files")
	f.BoolVar(&s.dryRun, "dry-run", false, "Print files that would be uploaded without actually saving")
	f.BoolVar(&s.uploadAssets, "assets", false, "Also upload assets from the "+assets.DefaultAssetsDir+" directory")

	return cmd
}

func (s *Save) Run(cmd *cobra.Command, args []string) {
	currentDir, err := os.Getwd()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get current directory").WithReason(err).Done()
	}

	fullPath := s.directory
	if !filepath.IsAbs(s.directory) {
		fullPath = filepath.Join(currentDir, s.directory)
	}

	projectId := s.projectId.GetFlagOrDie(fullPath)

	files, err := utils.CollectFiles(fullPath, s.dryRun)
	if err != nil {
		utils.NewExitError().WithMessage("failed to collect local files").WithReason(err).Done()
	}

	// If dry-run, print files and exit
	if s.dryRun {
		fmt.Println("Files that would be uploaded:")
		for _, file := range files {
			fmt.Println("  " + file.Filename)
		}

		return
	}

	payload := saveRequest{Files: files, Labels: map[string]string{}}

	requestPath := "o/:organisation/projects/" + projectId + "/files"

	err = s.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		WithPath(requestPath).
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to save files").WithReason(err).Done()
	}

	if s.uploadAssets {
		s.syncAssets(projectId, fullPath)
	}
}

func (s *Save) syncAssets(projectId, fullPath string) {
	orgId := s.configFactory.Context.OrganisationId
	assetDir := filepath.Join(fullPath, assets.DefaultAssetsDir)

	assetFiles, err := assets.CollectAssetFiles(assetDir)
	if err != nil {
		utils.NewExitError().WithMessage("failed to collect asset files").WithReason(err).Done()
	}

	if len(assetFiles) == 0 {
		fmt.Println("No asset files found in " + assets.DefaultAssetsDir)
		return
	}

	if s.dryRun {
		fmt.Println("Assets that would be uploaded:")
		for _, f := range assetFiles {
			fmt.Println("  " + filepath.Base(f))
		}
		return
	}

	for _, f := range assetFiles {
		fmt.Printf("Uploading asset %q...\n", filepath.Base(f))
		if uploadErr := assets.UploadAssetFile(s.configFactory, orgId, projectId, f, "research/documents"); uploadErr != nil {
			utils.NewExitError().WithMessage("failed to upload asset").WithReason(uploadErr).Done()
		}
		fmt.Printf("Successfully uploaded asset %q\n", filepath.Base(f))
	}
}
