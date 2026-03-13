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

package skills

import (
	"archive/zip"
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

//go:generate rm -rf ./skills
//go:generate cp -R ../../../skills ./skills
//go:embed skills/*
var f embed.FS

const skillsDirectory = "coding-versori-sdk"

type download struct {
	configFactory *config.ConfigFactory
	directory     string
	latest        bool
	agent         bool
}

func NewDownload(c *config.ConfigFactory) *cobra.Command {
	d := &download{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download AI agent skills",
		Long: `Download the AI agent skills to a local directory.

By default, this command extracts the skills that are shipped with the current CLI version,
requiring no network connection.

If the --latest flag is provided, it will attempt to download the latest version of the
skills directly from the main branch of the GitHub repository.`,
		Run: d.Run,
	}

	f := cmd.Flags()
	f.StringVarP(&d.directory, "directory", "d", "./skills", "Directory to save the skills into. It will append the skill name to the directory.")
	f.BoolVar(&d.latest, "latest", false, "Download the latest version of the skills")
	f.BoolVar(&d.agent, "agent", false, "Combine skills into a single AGENTS.md file")

	return cmd
}

func (d *download) Run(cmd *cobra.Command, args []string) {
	var files map[string][]byte

	if d.latest {
		files = d.collectZipFiles()
	} else {
		files = d.collectEmbeddedFiles()
	}

	if d.agent {
		d.combineAndWrite(files)
	} else {
		d.writeFiles(files)
	}
}

func (d *download) collectEmbeddedFiles() map[string][]byte {
	files := make(map[string][]byte)
	err := fs.WalkDir(f, "skills", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if d.agent && !strings.HasSuffix(entry.Name(), ".md") {
			return nil
		}

		rel, err := filepath.Rel("skills", path)
		if err != nil {
			return err
		}

		data, err := f.ReadFile(path)
		if err != nil {
			return err
		}

		files[rel] = data
		return nil
	})

	if err != nil {
		utils.NewExitError().WithMessage("failed to read embedded skills").WithReason(err).Done()
	}

	return files
}

func (d *download) collectZipFiles() map[string][]byte {
	resp, err := http.Get("https://github.com/versori/cli/archive/refs/heads/main.zip")
	if err != nil {
		utils.NewExitError().WithMessage("failed to download latest skills").WithReason(err).Done()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.NewExitError().WithMessage(fmt.Sprintf("failed to download latest skills: unexpected status code from github: %s", resp.Status)).Done()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.NewExitError().WithMessage("failed to read downloaded skills zip").WithReason(err).Done()
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		utils.NewExitError().WithMessage("failed to parse downloaded skills zip file").WithReason(err).Done()
	}

	prefix := "cli-main/skills/"
	files := make(map[string][]byte)

	for _, file := range zipReader.File {
		if !strings.HasPrefix(file.Name, prefix) {
			continue
		}

		if file.FileInfo().IsDir() {
			continue
		}

		if d.agent && !strings.HasSuffix(file.Name, ".md") {
			continue
		}

		relPath := strings.TrimPrefix(file.Name, prefix)

		rc, err := file.Open()
		if err != nil {
			utils.NewExitError().WithMessage("failed to open skill file from zip").WithReason(err).Done()
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			utils.NewExitError().WithMessage("failed to read skill file from zip").WithReason(err).Done()
		}

		files[relPath] = data
	}

	return files
}

func (d *download) writeFiles(files map[string][]byte) {
	dirName := filepath.Join(d.directory, skillsDirectory)
	if err := os.MkdirAll(dirName, 0755); err != nil {
		utils.NewExitError().WithMessage("failed to create directory").WithReason(err).Done()
	}

	for relPath, data := range files {
		target := filepath.Join(dirName, relPath)

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			utils.NewExitError().WithMessage("failed to create directory for skill file").WithReason(err).Done()
		}

		if err := os.WriteFile(target, data, 0644); err != nil {
			utils.NewExitError().WithMessage("failed to write skill file").WithReason(err).Done()
		}
	}
}

func (d *download) combineAndWrite(files map[string][]byte) {
	if err := os.MkdirAll(d.directory, 0755); err != nil {
		utils.NewExitError().WithMessage("failed to create directory").WithReason(err).Done()
	}

	var paths []string
	for path := range files {
		paths = append(paths, path)
	}

	sort.Slice(paths, func(i, j int) bool {
		nameI := filepath.Base(paths[i])
		nameJ := filepath.Base(paths[j])

		rank := func(name string) int {
			if strings.EqualFold(name, "readme.md") {
				return 0
			}
			if strings.EqualFold(name, "skill.md") || strings.EqualFold(name, "skills.md") {
				return 1
			}
			return 2
		}

		rankI := rank(nameI)
		rankJ := rank(nameJ)

		if rankI != rankJ {
			return rankI < rankJ
		}
		return paths[i] < paths[j]
	})

	var combinedContent bytes.Buffer

	for _, path := range paths {
		content := files[path]
		combinedContent.WriteString(fmt.Sprintf("\n\n<!-- BEGIN %s -->\n\n", path))
		combinedContent.Write(content)
		combinedContent.WriteString(fmt.Sprintf("\n\n<!-- END %s -->\n", path))
	}

	target := filepath.Join(d.directory, "AGENTS.md")
	err := os.WriteFile(target, combinedContent.Bytes(), 0644)
	if err != nil {
		utils.NewExitError().WithMessage("failed to write AGENTS.md file").WithReason(err).Done()
	}
}
