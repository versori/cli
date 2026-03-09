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

package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	v1 "github.com/versori/cli/pkg/api/v1"
)

// CollectFiles walks the provided fullPath directory and returns files suitable for upload.
// It respects .gitignore rules in that directory. If dryRun is true, file contents are omitted.
func CollectFiles(fullPath string, dryRun bool) ([]v1.File, error) {
	matcher := NewChecker()

	gitignorePath := filepath.Join(fullPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		// .gitignore exists, use it
		if err := matcher.LoadFile(fullPath); err != nil {
			return nil, fmt.Errorf("failed to load .gitignore: %w", err)
		}
	}

	files := make([]v1.File, 0, 128)
	err := filepath.WalkDir(fullPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if the file should be ignored by .gitignore
		if matcher != nil {
			fileInfo, statErr := entry.Info()
			if statErr != nil {
				return statErr
			}

			if matcher.Match(path, fileInfo) {
				if entry.IsDir() {
					return filepath.SkipDir
				}

				return nil // skip this file
			}
		}

		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}

			return nil
		}

		rel, relErr := filepath.Rel(fullPath, path)
		if relErr != nil {
			return relErr
		}

		if dryRun {
			files = append(files, v1.File{Filename: filepath.ToSlash(rel), Content: ""})

			return nil
		}

		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("failed to read file %s: %w", path, readErr)
		}

		files = append(files, v1.File{Filename: filepath.ToSlash(rel), Content: string(b)})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
