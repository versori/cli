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
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type Create struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	dryRun        bool
	name          string
	description   string
	directory     string
}

func NewCreate(c *config.ConfigFactory) *cobra.Command {
	p := &Create{configFactory: c}

	cmd := &cobra.Command{
		Use:     "create [--project <project-id>] [--directory <source_directory> [--name <version-name>] [--description <version-description>] [--dry-run]",
		Short:   "Creates a new version for a project with the files from --directory",
		Aliases: []string{"push"},
		Run:     p.Run,
	}

	flags := cmd.Flags()
	p.projectId.SetFlag(flags)
	flags.StringVarP(&p.name, "name", "n", "", "Name of the new version.")
	flags.StringVar(&p.description, "description", "", "Description of the new version.")
	flags.BoolVar(&p.dryRun, "dry-run", false, "Print files that would be uploaded without actually pushing.")
	flags.StringVarP(&p.directory, "directory", "d", ".", "Directory containing the new version.")

	return cmd
}

func (p *Create) Run(_ *cobra.Command, _ []string) {
	projectId := p.projectId.GetProjectIDFromDir(".")
	if projectId == "" {
		selectProject(p.configFactory, &projectId)
	}

	if p.name == "" {
		editor := elements.NewEditor("Version name:", false)
		err := editor.Edit(&p.name)
		if err != nil {
			utils.NewExitError().WithMessage("failed to get version name from editor").WithReason(err).Done()
		}
	}
	if p.description == "" {
		editor := elements.NewEditor("Version description:", true)
		err := editor.Edit(&p.description)
		if err != nil {
			utils.NewExitError().WithMessage("failed to get version description from editor").WithReason(err).Done()
		}
	}

	currentDir, err := os.Getwd()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get current working directory").WithReason(err).Done()
	}

	fullPath := p.directory
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(currentDir, fullPath)
	}

	files, err := utils.CollectFiles(fullPath, p.dryRun)
	if err != nil {
		utils.NewExitError().WithMessage("failed to collect local files").WithReason(err).Done()
	}

	if p.dryRun {
		fmt.Printf("New version %s for project %s would be created with files:\n", p.name, projectId)
		for _, file := range files {
			fmt.Println("  " + file.Filename)
		}

		return
	}

	reqBody := v1.CreateProjectVersion{
		Name:        p.name,
		Description: p.description,
		Files:       files,
	}

	requestPath := "o/:organisation/projects/" + projectId + "/versions"

	resp := v1.ProjectVersion{}
	err = p.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		WithPath(requestPath).
		Into(&resp).
		JSONBody(reqBody).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to push new version").WithReason(err).Done()
	}

	versionSummary := VersionSummary{
		ID:          resp.ID.String(),
		Name:        resp.Name,
		Description: resp.Description,
		State:       string(resp.State),
	}

	p.configFactory.Print(versionSummary)
}
