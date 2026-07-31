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
	"sort"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/cmd/projects/assets"
	"github.com/versori/cli/pkg/utils"
	"github.com/versori/cli/pkg/versorifile"
)

type Sync struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	directory     string
	dryRun        bool
	assets        bool
	confirm       bool
}

func NewSync(c *config.ConfigFactory) *cobra.Command {
	s := &Sync{configFactory: c}

	cmd := &cobra.Command{
		Use:                   "sync [--project <project-id>] [--directory <directory>]",
		DisableFlagsInUseLine: true,
		Long: `Sync pulls the project files to the local directory. The --project flag is only required the first time you sync a project.

Sync runs in dry-run mode by default and only prints what would be created, updated, or deleted. Pass --confirm to perform the real sync (which overwrites local changes and re-pins .versori).

WARNING: When --confirm is set, this will overwrite any local changes and delete any local files that are not part of the remote project.`,
		Short: "Sync pulls the project files to the local directory. Dry-run by default; pass --confirm to actually write.",
		Run:   s.Run,
	}

	f := cmd.Flags()
	s.projectId.SetFlag(f)
	f.StringVarP(&s.directory, "directory", "d", ".", "The directory to download the project files into")
	f.BoolVar(&s.dryRun, "dry-run", false, "Force dry-run (the default when --confirm is omitted). Kept for explicitness; if both --dry-run and --confirm are set, --dry-run wins.")
	f.BoolVar(&s.assets, "assets", false, "Also sync project assets, removing any that are no longer part of the project from the "+assets.DefaultAssetsDir+" directory")
	f.BoolVar(&s.confirm, "confirm", false, "Perform the actual sync. Without this flag, sync only prints what would change (dry-run).")

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

	// Dry-run is the default; --confirm opts in to the real sync. --dry-run
	// stays available as an explicit toggle and wins on conflict to keep the
	// safer behaviour.
	defaultedToDryRun := !s.dryRun && !s.confirm
	s.dryRun = s.dryRun || !s.confirm

	if defaultedToDryRun {
		fmt.Fprintln(os.Stderr, "No --confirm flag: running in dry-run mode. Re-run with --confirm to actually sync.")
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

	existing, dirs := getExistingFiles(fullPath)

	// Write/update all project files
	var created, updated []string
	unchanged := 0

	for _, f := range project.CurrentFiles.Files {
		rel, action := updateFile(f, fullPath, existing, s.dryRun)

		switch action {
		case actionCreate:
			created = append(created, rel)
		case actionUpdate:
			updated = append(updated, rel)
		case actionUnchanged:
			unchanged++
		}
	}

	// Delete any extra files
	if !s.dryRun {
		for rel := range existing {
			abs := filepath.Join(fullPath, rel)
			_ = os.Remove(abs)
		}

		pruneEmptyDirs(fullPath, dirs)
	}

	if s.dryRun {
		deleted := make([]string, 0, len(existing))
		for rel := range existing {
			deleted = append(deleted, rel)
		}

		printPlan(created, updated, deleted, unchanged)
	}

	if s.assets {
		s.syncAssets(projectId, fullPath)
	}

	if s.dryRun {
		fmt.Fprintln(os.Stderr, "Dry-run complete. Re-run with --confirm to apply these changes.")
		return
	}

	versoriPath := filepath.Join(fullPath, ".versori")
	if err := versorifile.Write(versoriPath, &versorifile.VersoriFile{ProjectId: projectId, Context: s.configFactory.Context.Name}); err != nil {
		utils.NewExitError().WithMessage("failed to write .versori").WithReason(err).Done()
	}
}

func (s *Sync) syncAssets(projectId, fullPath string) {
	resp, err := assets.ListAssets(s.configFactory, projectId)
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

// fileAction describes what a sync did (or would do) to a single project file.
type fileAction int

const (
	actionUnchanged fileAction = iota
	actionCreate
	actionUpdate
)

// printPlan reports only the files a sync would actually touch. Files whose
// local contents already match the remote project are collapsed into a count so
// the plan stays readable on projects where most files are up to date.
func printPlan(created, updated, deleted []string, unchanged int) {
	sort.Strings(created)
	sort.Strings(updated)
	sort.Strings(deleted)

	for _, group := range []struct {
		label string
		files []string
	}{
		{"Files that would be created:", created},
		{"Files that would be updated:", updated},
		{"Files that would be deleted:", deleted},
	} {
		if len(group.files) == 0 {
			continue
		}

		fmt.Println(group.label)

		for _, rel := range group.files {
			fmt.Println("  " + rel)
		}
	}

	if len(created) == 0 && len(updated) == 0 && len(deleted) == 0 {
		fmt.Printf("Already up to date (%d files unchanged).\n", unchanged)

		return
	}

	if unchanged > 0 {
		fmt.Printf("%d file(s) already up to date.\n", unchanged)
	}
}

func updateFile(f v1.File, fullPath string, existing map[string]struct{}, dryRun bool) (string, fileAction) {
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

	newContent := []byte(f.Content)
	action := actionUpdate

	// Compare against the local file so that both modes agree on what is
	// actually affected: a file whose contents already match is left alone by
	// the real sync, so a dry-run must not report it either.
	old, readErr := os.ReadFile(dest)

	switch {
	case readErr == nil:
		if bytes.Equal(old, newContent) {
			action = actionUnchanged
		}
	case os.IsNotExist(readErr):
		action = actionCreate
	default:
		// Only ignore not-exist errors; fail on others
		utils.NewExitError().WithMessage("failed to read existing file " + dest).WithReason(readErr).Done()
	}

	if !dryRun && action != actionUnchanged {
		// make sure directory exists
		if mkErr := os.MkdirAll(filepath.Dir(dest), 0755); mkErr != nil {
			utils.NewExitError().WithMessage("failed to create directory for " + dest).WithReason(mkErr).Done()
		}

		if writeErr := os.WriteFile(dest, newContent, 0o600); writeErr != nil {
			utils.NewExitError().WithMessage("failed to write file " + dest).WithReason(writeErr).Done()
		}
	}

	// mark as handled so it won't be deleted
	if relKey, relErr := filepath.Rel(fullPath, dest); relErr == nil {
		delete(existing, relKey)
	}

	return rel, action
}

// getExistingFiles returns the set of local files (relative to fullPath) that
// are candidates for deletion, plus the directories encountered along the way.
// Directories are kept separate because they are only removed once they are
// left empty, so they must not appear in the list of affected files.
func getExistingFiles(fullPath string) (map[string]struct{}, []string) {
	// Build a set of existing files (relative to fullPath) that could be deleted later
	existing := map[string]struct{}{}

	var dirs []string

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

		if d.IsDir() {
			dirs = append(dirs, rel)

			return nil
		}

		existing[rel] = struct{}{}

		return nil
	})
	if err != nil {
		utils.NewExitError().WithMessage("failed to walk directory").WithReason(err).Done()
	}

	return existing, dirs
}

// pruneEmptyDirs removes directories left empty after orphaned files were
// deleted, deepest first so that parents become removable in the same pass.
// Directories which still hold files are left alone: os.Remove fails on them
// and the error is intentionally ignored.
func pruneEmptyDirs(fullPath string, dirs []string) {
	sort.Slice(dirs, func(i, j int) bool {
		return strings.Count(dirs[i], string(os.PathSeparator)) > strings.Count(dirs[j], string(os.PathSeparator))
	})

	for _, rel := range dirs {
		_ = os.Remove(filepath.Join(fullPath, rel))
	}
}
