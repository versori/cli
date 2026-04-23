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

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type FileSummary struct {
	Filename string `json:"filename"`
	Size     int    `json:"size"`
	Content  string `json:"content"`
}

func init() {
	utils.RegisterResource(FileSummary{}, []string{"Filename", "Size"})
}

type files struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	version       string
}

func NewFiles(c *config.ConfigFactory) *cobra.Command {
	f := &files{configFactory: c}

	cmd := &cobra.Command{
		Use:   "files [filename]",
		Short: "List or read files from a project",
		Long: `List files in a project, or print a single file's content to stdout.

With no arguments, lists files. Default -o table shows filename and size;
-o json and -o yaml include file contents too.

With a filename, writes that file's content verbatim to stdout so it can be
piped to other tools (jq, less, grep, etc.).`,
		Args: cobra.MaximumNArgs(1),
		Run:  f.Run,
	}

	f.projectId.SetFlag(cmd.Flags())
	cmd.Flags().StringVar(&f.version, "version", "", "Read files from a specific version id instead of the current files")

	return cmd
}

func (f *files) Run(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get current directory").WithReason(err).Done()
	}
	projectId := f.projectId.GetFlagOrDie(cwd)

	var filename string
	if len(args) == 1 {
		filename = args[0]
	}

	path := "o/:organisation/projects/" + projectId + "/files"
	if f.version != "" {
		path = "o/:organisation/projects/" + projectId + "/versions/" + f.version + "/files"
	}

	result := v1.Files{}
	err = f.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&result).
		WithPath(path).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get project files").WithReason(err).Done()
	}

	if filename != "" {
		for _, file := range result.Files {
			if file.Filename == filename {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), file.Content)
				return
			}
		}
		utils.NewExitError().WithMessage(fmt.Sprintf("file %q not found in project", filename)).Done()
	}

	summaries := make([]FileSummary, 0, len(result.Files))
	for _, file := range result.Files {
		summaries = append(summaries, FileSummary{
			Filename: file.Filename,
			Size:     len(file.Content),
			Content:  file.Content,
		})
	}
	f.configFactory.Print(summaries)
}
