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
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/cmd/projects/assets"
	"github.com/versori/cli/pkg/utils"
)

type Sync struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	directory     string
	dryRun        bool
	assets        bool
}

func NewSync(c *config.ConfigFactory) *cobra.Command {
	s := &Sync{configFactory: c}

	cmd := &cobra.Command{
		Use:                   "sync [--project <project-id>] [--directory <directory>]",
		DisableFlagsInUseLine: true,
		Long: `Sync pulls the project files to the local directory. The --project flag is only required the first time you sync a project.
WARNING: This will overwrite any local changes`,
		Short: "Sync pulls the project files to the local directory. WARNING: This will overwrite any local changes",
		Run:   s.Run,
	}

	f := cmd.Flags()
	s.projectId.SetFlag(f)
	f.StringVarP(&s.directory, "directory", "d", ".", "The directory to download the project files into")
	f.BoolVar(&s.dryRun, "dry-run", false, "Print files that would be created/updated/deleted without actually syncing")
	f.BoolVar(&s.assets, "assets", false, "Also sync project assets, removing any that are no longer part of the project from the "+assets.DefaultAssetsDir+" directory")

	return cmd
}

func (s *Sync) Run(cmd *cobra.Command, args []string) {
	currentDir, err := os.Getwd()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get current directory").WithReason(err).Done()
	}

	fullPath := s.directory
	if !filepath.IsAbs(s.directory) {
		fullPath = filepath.Join(currentDir, s.directory)
	}

	err = os.MkdirAll(fullPath, 0755)
	if err != nil {
		utils.NewExitError().WithMessage("failed to create directory").WithReason(err).Done()
	}

	projectId := s.projectId.GetFlagOrDie(fullPath)

	project := v1.Project{}

	err = s.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&project).
		WithPath("o/:organisation/projects/" + projectId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get project").WithReason(err).Done()
	}

	existing := getExistingFiles(fullPath)

	// Write/update all project files
	for _, f := range project.CurrentFiles.Files {
		updateFile(f, fullPath, existing, s.dryRun)
	}

	// Delete any extra files
	if len(existing) > 0 {
		if s.dryRun {
			fmt.Println("Files that would be deleted:")
			for rel := range existing {
				fmt.Println("  " + rel)
			}
		} else {
			for rel := range existing {
				abs := filepath.Join(fullPath, rel)
				_ = os.Remove(abs)
			}
		}
	}

	if s.assets {
		s.syncAssets(projectId, fullPath)
	}

	if !s.dryRun {
		versoriPath := filepath.Join(fullPath, ".versori")
		if err := flags.WriteVersoriConfig(versoriPath, &flags.VersoriFile{ProjectId: projectId, Context: s.configFactory.Context.Name}); err != nil {
			utils.NewExitError().WithMessage("failed to write .versori").WithReason(err).Done()
		}
	}
}

func (s *Sync) syncAssets(projectId, fullPath string) {
	orgId := s.configFactory.Context.OrganisationId

	resp, err := assets.ListAssets(s.configFactory, orgId, projectId)
	if err != nil {
		utils.NewExitError().WithMessage("failed to list assets").WithReason(err).Done()
	}

	assetPath := filepath.Join(fullPath, assets.DefaultAssetsDir)

	// Build set of remote asset names
	remoteAssets := make(map[string]struct{}, len(resp.Assets))
	for _, a := range resp.Assets {
		remoteAssets[a.Name] = struct{}{}
	}

	// Remove local assets that are no longer in the project
	if entries, readErr := os.ReadDir(assetPath); readErr == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if _, exists := remoteAssets[entry.Name()]; !exists {
				localFile := filepath.Join(assetPath, entry.Name())
				if s.dryRun {
					fmt.Println("Would delete asset: " + entry.Name())
				} else {
					_ = os.Remove(localFile)
				}
			}
		}
	}

	// Download all current assets
	for _, a := range resp.Assets {
		if a.DownloadURL == "" {
			fmt.Printf("Skipping asset %q: no download URL\n", a.Name)
			continue
		}

		if s.dryRun {
			fmt.Println("Would download asset: " + a.Name)
			continue
		}

		if dlErr := assets.DownloadAssetToFile(a.DownloadURL, a.Name, assetPath); dlErr != nil {
			utils.NewExitError().WithMessage("failed to download asset " + a.Name).WithReason(dlErr).Done()
		}
	}
}

func updateFile(f v1.File, fullPath string, existing map[string]struct{}, dryRun bool) {
	// sanitize and ensure path stays within fullPath
	rel := filepath.Clean(f.Filename)
	if strings.Contains(rel, "..") || filepath.IsAbs(rel) {
		utils.NewExitError().
			WithMessage("invalid file path in project").
			WithReason(fmt.Errorf("invalid file path in project: %s", f.Filename)).
			Done()
	}

	dest := filepath.Join(fullPath, rel)
	// ensure dest is still under fullPath after cleaning
	cleanDest := filepath.Clean(dest)
	if !strings.HasPrefix(cleanDest+string(os.PathSeparator), filepath.Clean(fullPath)+string(os.PathSeparator)) && cleanDest != filepath.Clean(fullPath) {
		utils.NewExitError().
			WithMessage("refusing to write outside sync directory").
			WithReason(fmt.Errorf("refusing to write outside sync directory: %s", cleanDest)).
			Done()
	}

	if dryRun {
		// In dry-run mode, just check if file exists and print what would happen
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			fmt.Println("Would create: " + rel)
		} else {
			fmt.Println("Would update: " + rel)
		}
	} else {
		// make sure directory exists
		if mkErr := os.MkdirAll(filepath.Dir(dest), 0755); mkErr != nil {
			utils.NewExitError().WithMessage("failed to create directory for " + dest).WithReason(mkErr).Done()
		}

		newContent := []byte(f.Content)
		if old, readErr := os.ReadFile(dest); readErr == nil {
			if !bytes.Equal(old, newContent) {
				if writeErr := os.WriteFile(dest, newContent, 0o600); writeErr != nil {
					utils.NewExitError().WithMessage("failed to update file " + dest).WithReason(writeErr).Done()
				}
			}
		} else {
			// Only ignore not-exist errors; fail on others
			if !os.IsNotExist(readErr) {
				utils.NewExitError().WithMessage("failed to read existing file " + dest).WithReason(readErr).Done()
			}
			if writeErr := os.WriteFile(dest, newContent, 0o600); writeErr != nil {
				utils.NewExitError().WithMessage("failed to create file " + dest).WithReason(writeErr).Done()
			}
		}
	}

	// mark as handled so it won't be deleted
	if relKey, relErr := filepath.Rel(fullPath, dest); relErr == nil {
		delete(existing, relKey)
	}
}

func getExistingFiles(fullPath string) map[string]struct{} {
	// Build a set of existing files (relative to fullPath) that could be deleted later
	existing := map[string]struct{}{}

	checker := utils.NewChecker()

	gitignorePath := filepath.Join(fullPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		// .gitignore exists, use it
		if err := checker.LoadFile(fullPath); err != nil {
			utils.NewExitError().WithMessage("failed to load .gitignore").WithReason(err).Done()
		}
	}

	err := filepath.WalkDir(fullPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// skip root dir so we don't delete the whole project locally
		if d.IsDir() && path == fullPath {
			return nil
		}

		// Check if file should be ignored by .gitignore
		if checker != nil {
			fileInfo, statErr := d.Info()
			if statErr != nil {
				return statErr
			}

			if checker.Match(path, fileInfo) {
				if d.IsDir() {
					return filepath.SkipDir
				}

				return nil
			}
		}

		rel, relErr := filepath.Rel(fullPath, path)
		if relErr != nil {
			return relErr
		}

		// never delete the context file or assets directory
		if rel == ".versori" {
			return nil
		}

		if rel == assets.DefaultAssetsDir || strings.HasPrefix(rel, assets.DefaultAssetsDir+string(os.PathSeparator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		existing[rel] = struct{}{}

		return nil
	})
	if err != nil {
		utils.NewExitError().WithMessage("failed to walk directory").WithReason(err).Done()
	}

	return existing
}
