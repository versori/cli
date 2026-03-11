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
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type Pull struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	directory     string
	dryRun        bool
	version       string
}

func NewPull(c *config.ConfigFactory) *cobra.Command {
	p := &Pull{configFactory: c}

	cmd := &cobra.Command{
		Use:   "pull [--directory <target-directory>] [--project <project-id>] [--version <version-id>] [--dry-run]",
		Short: "Pull files from a version into the target directory",
		Run:   p.Run,
	}

	flags := cmd.Flags()
	p.projectId.SetFlag(flags)
	flags.StringVar(&p.version, "version", "", "ID of the version. If not given, you will be prompted to select one.")
	flags.BoolVar(&p.dryRun, "dry-run", false, "Print files that would be uploaded without actually pushing.")
	flags.StringVarP(&p.directory, "directory", "d", ".", "Directory containing the versions to be uploaded")

	return cmd
}

func (p *Pull) Run(_ *cobra.Command, _ []string) {
	var err error

	projectId := p.projectId.GetProjectIDFromDir(".")
	if projectId == "" {
		selectProject(p.configFactory, &projectId)
	}

	if p.version == "" {
		selectVersion(p.configFactory, projectId, &p.version)
	}

	var files v1.Files
	err = p.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&files).
		WithPath("o/:organisation/projects/" + projectId + "/versions/" + p.version + "/files").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get version files").WithReason(err).Done()
	}

	currentDir, err := os.Getwd()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get current working directory").WithReason(err).Done()
	}

	fullPath := p.directory
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(currentDir, fullPath)
	}

	if p.dryRun {
		fmt.Printf("The following files would be pulled into the target directory: %s\n", fullPath)
		for _, file := range files.Files {
			fmt.Println("  " + file.Filename)
		}

		return
	}

	_, err = os.Stat(fullPath)
	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(fullPath, 0755)
		if err != nil {
			utils.NewExitError().WithMessage("failed to create target directory").WithReason(err).Done()
		}
	} else if errors.Is(err, os.ErrNotExist) {
		utils.NewExitError().WithMessage("target directory does not exist; use --create to create it").Done()
	}

	for _, f := range files.Files {
		fname := filepath.Join(fullPath, f.Filename)
		// Ensure parent directories exist
		err = os.MkdirAll(filepath.Dir(fname), 0755)
		if err != nil {
			utils.NewExitError().WithMessage(fmt.Sprintf("failed to create parent directories for file %s", f.Filename)).WithReason(err).Done()
		}

		err = os.WriteFile(fname, []byte(f.Content), 0o600)
		if err != nil {
			utils.NewExitError().WithMessage(fmt.Sprintf("failed to write file %s", f.Filename)).WithReason(err).Done()
		}
	}
}
