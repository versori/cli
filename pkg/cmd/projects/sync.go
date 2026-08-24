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
	version       string
	noPin         bool
}

func NewSync(c *config.ConfigFactory) *cobra.Command {
	s := &Sync{configFactory: c}

	cmd := &cobra.Command{
		Use:                   "sync [--project <project-id>] [--directory <directory>] [--version <version-id>] [--no-pin]",
		DisableFlagsInUseLine: true,
		Long: `Sync pulls the project files to the local directory. The --project flag is only required the first time you sync a project.

Sync runs in dry-run mode by default and only prints what would be created, updated, or deleted. Pass --confirm to perform the real sync (which overwrites local changes and re-pins .versori).

Pass --version <id> to sync a specific version's files instead of the project's current files. Pass --no-pin to perform the sync without writing .versori, which is how a throwaway comparison checkout avoids claiming the directory.

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
	f.StringVar(&s.version, "version", "", "Sync the files from a specific version id instead of the project's current files. Assets are always the project's current assets.")
	f.BoolVar(&s.noPin, "no-pin", false, "Do not write .versori into the target directory. An existing .versori is left untouched.")

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

	versionId, ok := normalizeVersionId(s.version)
	if !ok {
		utils.NewExitError().WithMessage("--version requires a version id").Done()
	}

	var sourceFiles []v1.File

	if versionId == "" {
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

		sourceFiles = project.CurrentFiles.Files
	} else {
		files := v1.Files{}

		err = s.configFactory.
			NewRequest().
			WithMethod(http.MethodGet).
			Into(&files).
			WithPath(versionFilesPath(projectId, versionId)).
			Do()
		if err != nil {
			utils.NewExitError().WithMessage("failed to get version files").WithReason(err).Done()
		}

		sourceFiles = files.Files
	}

	existing, dirs := getExistingFiles(fullPath)

	// Write/update all project files
	var created, updated, unchanged []string

	for _, f := range sourceFiles {
		rel, action := updateFile(f, fullPath, existing, s.dryRun)

		switch action {
		case actionCreate:
			created = append(created, rel)
		case actionUpdate:
			updated = append(updated, rel)
		case actionUnchanged:
			unchanged = append(unchanged, rel)
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

	if !shouldWritePin(s.dryRun, s.noPin) {
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

// minCollapse is the number of files a directory must contribute before it is
// worth collapsing: rewriting a lone "src/a.ts" as "src/ (1 file)" only hides
// its name.
const minCollapse = 2

// planEntry is one line of the plan: either a single file, or a directory
// standing in for every file beneath it.
type planEntry struct {
	path  string
	count int
}

// printPlan reports only the files a sync would actually touch. Files whose
// local contents already match the remote project are collapsed into a count so
// the plan stays readable on projects where most files are up to date.
func printPlan(created, updated, deleted, unchanged []string) {
	// Every path the sync knows about, used as the denominator when deciding
	// whether a directory is affected in its entirety.
	totals := countByDir(created, updated, deleted, unchanged)

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

		for _, e := range collapseByDir(group.files, totals) {
			if e.count == 0 {
				fmt.Println("  " + e.path)

				continue
			}

			fmt.Printf("  %s/ (%s)\n", e.path, pluralFiles(e.count))
		}
	}

	if len(created) == 0 && len(updated) == 0 && len(deleted) == 0 {
		fmt.Printf("Already up to date (%s unchanged).\n", pluralFiles(len(unchanged)))

		return
	}

	if len(unchanged) > 0 {
		fmt.Printf("%s already up to date.\n", pluralFiles(len(unchanged)))
	}
}

func pluralFiles(n int) string {
	if n == 1 {
		return "1 file"
	}

	return fmt.Sprintf("%d files", n)
}

// collapseByDir folds each run of files into the highest directory that this
// group owns outright, mirroring the way git reports a wholly-untracked
// directory as a single entry. A directory is only collapsed when every file
// the sync knows about inside it belongs to this same group, so a collapsed
// line can never hide a file with a different fate. The repository root is
// never collapsed, otherwise a first-time sync would report nothing but ".".
func collapseByDir(files []string, totals map[string]int) []planEntry {
	mine := countByDir(files)

	collapsed := make(map[string]int)

	var entries []planEntry

	for _, rel := range files {
		// Ancestors run deepest-first, so walk backwards to prefer the
		// shallowest directory this group fully owns.
		anc := ancestors(rel)

		target := ""

		for i := len(anc) - 1; i >= 0; i-- {
			d := anc[i]
			if mine[d] == totals[d] && totals[d] >= minCollapse {
				target = d

				break
			}
		}

		if target == "" {
			entries = append(entries, planEntry{path: rel})

			continue
		}

		collapsed[target] = mine[target]
	}

	for dir, count := range collapsed {
		entries = append(entries, planEntry{path: dir, count: count})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })

	return entries
}

// countByDir tallies how many of the given files live under each directory, at
// every level of nesting.
func countByDir(groups ...[]string) map[string]int {
	counts := make(map[string]int)

	for _, files := range groups {
		for _, rel := range files {
			for _, d := range ancestors(rel) {
				counts[d]++
			}
		}
	}

	return counts
}

// ancestors lists the directories containing rel, deepest first, excluding the
// root itself.
func ancestors(rel string) []string {
	var out []string

	for dir := filepath.Dir(rel); dir != "." && dir != string(os.PathSeparator); dir = filepath.Dir(dir) {
		out = append(out, dir)
	}

	return out
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
	oldContent, readErr := os.ReadFile(dest)

	switch {
	case readErr == nil:
		if bytes.Equal(oldContent, newContent) {
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

// versionFilesPath is the API path holding a single version's files. It
// deliberately matches the path `projects files --version` uses, so the two
// commands can never drift onto different endpoints.
func versionFilesPath(projectId, versionId string) string {
	return "o/:organisation/projects/" + projectId + "/versions/" + versionId + "/files"
}

// normalizeVersionId trims a --version value and reports whether a version was
// actually requested. A value that is only whitespace is a mistake, not a
// request for the current files: sync fails rather than silently syncing
// something else.
func normalizeVersionId(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", raw == ""
	}

	return trimmed, true
}

// shouldWritePin reports whether this sync re-pins .versori. Dry-runs never
// wrote it, and --no-pin opts a real sync out; neither ever deletes a .versori
// that is already there.
func shouldWritePin(dryRun, noPin bool) bool {
	return !dryRun && !noPin
}
