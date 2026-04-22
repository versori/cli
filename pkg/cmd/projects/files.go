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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type FileSummary struct {
	Filename string `json:"filename"`
	Size     int    `json:"size"`
}

func init() {
	utils.RegisterResource(FileSummary{}, []string{"Filename", "Size"})
}

type files struct {
	configFactory *config.ConfigFactory
	all           bool
	version       string
}

func NewFiles(c *config.ConfigFactory) *cobra.Command {
	f := &files{configFactory: c}

	cmd := &cobra.Command{
		Use:   "files <project-id> [filename]",
		Short: "List or read files from a project. Pass in - as the project id to read it from stdin",
		Long: `List filenames in a project, or print a single file's content to stdout.

With no filename argument, lists filenames (honours -o table/json/yaml).
With a filename, writes that file's content verbatim to stdout so it can be
piped to other tools (jq, less, grep, etc.).
With --all, dumps all files including contents as JSON.`,
		Args: cobra.RangeArgs(1, 2),
		Run:  f.Run,
	}

	flags := cmd.Flags()
	flags.BoolVar(&f.all, "all", false, "Dump all files (filenames and contents) as JSON to stdout")
	flags.StringVar(&f.version, "version", "", "Read files from a specific version id instead of the current files")

	return cmd
}

func (f *files) Run(cmd *cobra.Command, args []string) {
	projectId := args[0]
	if projectId == "-" {
		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			utils.NewExitError().WithMessage("failed to read project id from stdin").WithReason(err).Done()
		}
		projectId = strings.TrimSpace(string(b))
	}

	var filename string
	if len(args) == 2 {
		filename = args[1]
	}

	if filename != "" && f.all {
		utils.NewExitError().WithMessage("cannot pass both a filename argument and --all").Done()
	}

	path := "o/:organisation/projects/" + projectId + "/files"
	if f.version != "" {
		path = "o/:organisation/projects/" + projectId + "/versions/" + f.version + "/files"
	}

	result := v1.Files{}
	err := f.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&result).
		WithPath(path).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get project files").WithReason(err).Done()
	}

	switch {
	case filename != "":
		for _, file := range result.Files {
			if file.Filename == filename {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), file.Content)
				return
			}
		}
		utils.NewExitError().WithMessage(fmt.Sprintf("file %q not found in project", filename)).Done()
	case f.all:
		data, err := json.Marshal(result)
		if err != nil {
			utils.NewExitError().WithMessage("failed to marshal files").WithReason(err).Done()
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	default:
		summaries := make([]FileSummary, 0, len(result.Files))
		for _, file := range result.Files {
			summaries = append(summaries, FileSummary{Filename: file.Filename, Size: len(file.Content)})
		}
		f.configFactory.Print(summaries)
	}
}
