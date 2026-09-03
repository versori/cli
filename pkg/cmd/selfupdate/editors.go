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

package selfupdate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type EditorKind string

const (
	EditorVSCode EditorKind = "VS Code"
	EditorCursor EditorKind = "Cursor"
)

type Editor struct {
	Kind EditorKind
	Path string
}

func ResolveEditors(vscodePath, cursorPath string) ([]Editor, error) {
	var editors []Editor

	for _, item := range []struct {
		kind EditorKind
		flag string
	}{
		{EditorVSCode, vscodePath},
		{EditorCursor, cursorPath},
	} {
		path, _, err := LookEditor(item.kind, item.flag)
		if err != nil {
			return nil, err
		}
		if path == "" {
			continue
		}
		editors = append(editors, Editor{Kind: item.kind, Path: path})
	}

	return editors, nil
}

func LookEditor(kind EditorKind, flagPath string) (path string, flagged bool, err error) {
	if flagPath != "" {
		if !isExecutable(flagPath) {
			return "", true, fmt.Errorf("%s CLI %q is missing or not executable", kind, flagPath)
		}
		return flagPath, true, nil
	}

	if name := editorBinName(kind); name != "" {
		found, lookErr := exec.LookPath(name)
		if lookErr == nil && isExecutable(found) {
			return found, false, nil
		}
	}

	for _, candidate := range editorWellKnown(kind) {
		if isExecutable(candidate) {
			return candidate, false, nil
		}
	}

	return "", false, nil
}

func editorBinName(kind EditorKind) string {
	switch kind {
	case EditorVSCode:
		return "code"
	case EditorCursor:
		return "cursor"
	default:
		return ""
	}
}

var editorWellKnown = wellKnownEditorPaths

func wellKnownEditorPaths(kind EditorKind) []string {
	switch runtime.GOOS {
	case "darwin":
		switch kind {
		case EditorVSCode:
			return []string{
				"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
				"/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code",
			}
		case EditorCursor:
			return []string{
				"/Applications/Cursor.app/Contents/Resources/app/bin/cursor",
			}
		}
	case "linux":
		switch kind {
		case EditorVSCode:
			return []string{
				"/usr/bin/code",
				"/usr/share/code/bin/code",
			}
		case EditorCursor:
			return []string{
				"/usr/bin/cursor",
				"/usr/share/cursor/bin/cursor",
			}
		}
	case "windows":
		localApp := os.Getenv("LOCALAPPDATA")
		if localApp == "" {
			return nil
		}
		switch kind {
		case EditorVSCode:
			return []string{
				filepath.Join(localApp, "Programs", "Microsoft VS Code", "bin", "code.cmd"),
			}
		case EditorCursor:
			return []string{
				filepath.Join(localApp, "Programs", "cursor", "resources", "app", "bin", "cursor.cmd"),
			}
		}
	}
	return nil
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}
