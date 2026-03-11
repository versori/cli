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
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
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

	return cmd
}

func (d *download) Run(cmd *cobra.Command, args []string) {
	if d.latest {
		d.downloadLatest()
		return
	}

	dirName := filepath.Join(d.directory, skillsDirectory)

	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		utils.NewExitError().WithMessage("failed to create directory").WithReason(err).Done()
	}

	err = fs.WalkDir(f, "skills", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel("skills", path)
		if err != nil {
			return err
		}

		target := filepath.Join(dirName, rel)

		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := f.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0644)
	})

	if err != nil {
		utils.NewExitError().WithMessage("failed to extract embedded skills").WithReason(err).Done()
	}
}

func (d *download) downloadLatest() {
	resp, err := http.Get("https://github.com/versori/cli/archive/refs/heads/main.zip")
	if err != nil {
		utils.NewExitError().WithMessage("failed to download latest skills").WithReason(err).Done()
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		utils.NewExitError().WithMessage("failed to download latest skills: unexpected status code from github: " + resp.Status).Done()
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

	for _, file := range zipReader.File {
		if !strings.HasPrefix(file.Name, prefix) {
			continue
		}

		relPath := strings.TrimPrefix(file.Name, prefix)
		if relPath == "" {
			continue
		}

		target := filepath.Join(d.directory, skillsDirectory, relPath)

		if file.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			utils.NewExitError().WithMessage("failed to create directory for skill file").WithReason(err).Done()
		}

		outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			utils.NewExitError().WithMessage("failed to create skill file").WithReason(err).Done()
		}

		rc, err := file.Open()
		if err != nil {
			_ = outFile.Close()
			utils.NewExitError().WithMessage("failed to open skill file from zip").WithReason(err).Done()
		}

		_, err = io.Copy(outFile, rc)
		_ = rc.Close()
		_ = outFile.Close()

		if err != nil {
			utils.NewExitError().WithMessage("failed to write skill file").WithReason(err).Done()
		}
	}
}
