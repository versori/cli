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
	"embed"
	_ "embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

//go:embed skills/*
var f embed.FS

type download struct {
	configFactory *config.ConfigFactory
	directory     string
}

func NewDownload(c *config.ConfigFactory) *cobra.Command {
	d := &download{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "download --directory <directory>",
		Short: "Download the skills",
		Run:   d.Run,
	}

	f := cmd.Flags()
	f.StringVarP(&d.directory, "directory", "d", "./skills", "Directory to save the skills into. It will append the skill name to the directory.")

	return cmd
}

func (d *download) Run(cmd *cobra.Command, args []string) {
	dirName := filepath.Join(d.directory, "coding-versori-sdk")

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
		utils.NewExitError().WithMessage("failed to download skills").WithReason(err).Done()
	}
}
