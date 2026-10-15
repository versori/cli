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
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveEditorsPATH(t *testing.T) {
	isolateWellKnown(t)

	dir := t.TempDir()
	codePath := writeFakeEditor(t, dir, "code")
	t.Setenv("PATH", dir)

	editors, err := ResolveEditors("", "")
	if err != nil {
		t.Fatal(err)
	}
	got := editorPath(editors, EditorVSCode)
	if got != codePath {
		t.Fatalf("VS Code path = %q; want %q (editors=%v)", got, codePath, editors)
	}
	if editorPath(editors, EditorCursor) != "" {
		t.Fatalf("Cursor should not resolve from isolated PATH: %v", editors)
	}
}

func TestResolveEditorsFlagMissing(t *testing.T) {
	isolateWellKnown(t)
	t.Setenv("PATH", t.TempDir())

	missing := filepath.Join(t.TempDir(), "no-such-code")
	_, err := ResolveEditors(missing, "")
	if err == nil {
		t.Fatal("ResolveEditors missing --vscode-path: want error")
	}

	_, err = ResolveEditors("", filepath.Join(t.TempDir(), "no-such-cursor"))
	if err == nil {
		t.Fatal("ResolveEditors missing --cursor-path: want error")
	}
}

func TestResolveEditorsFlagWinsOverPATH(t *testing.T) {
	isolateWellKnown(t)

	pathDir := t.TempDir()
	flagDir := t.TempDir()
	pathCode := writeFakeEditor(t, pathDir, "code")
	flagCode := writeFakeEditor(t, flagDir, "code")
	t.Setenv("PATH", pathDir)

	editors, err := ResolveEditors(flagCode, "")
	if err != nil {
		t.Fatal(err)
	}
	got := editorPath(editors, EditorVSCode)
	if got != flagCode {
		t.Fatalf("flag path = %q; want %q (PATH had %q, editors=%v)", got, flagCode, pathCode, editors)
	}
}

func TestLookEditorFlagNotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows treats existing files as executable")
	}
	isolateWellKnown(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "code")
	if err := os.WriteFile(path, []byte("not executable"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, flagged, err := LookEditor(EditorVSCode, path)
	if err == nil {
		t.Fatal("LookEditor non-executable flag: want error")
	}
	if !flagged {
		t.Fatal("LookEditor: flagged=false; want true")
	}
}

func TestWellKnownEditorPaths(t *testing.T) {
	vscode := wellKnownEditorPaths(EditorVSCode)
	cursor := wellKnownEditorPaths(EditorCursor)
	switch runtime.GOOS {
	case "darwin":
		wantVS := []string{
			"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
			"/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code",
		}
		wantCur := []string{
			"/Applications/Cursor.app/Contents/Resources/app/bin/cursor",
		}
		assertPaths(t, vscode, wantVS)
		assertPaths(t, cursor, wantCur)
	case "linux":
		assertPaths(t, vscode, []string{"/usr/bin/code", "/usr/share/code/bin/code"})
		assertPaths(t, cursor, []string{"/usr/bin/cursor", "/usr/share/cursor/bin/cursor"})
	}
}

func isolateWellKnown(t *testing.T) {
	t.Helper()
	orig := editorWellKnown
	editorWellKnown = func(EditorKind) []string { return nil }
	t.Cleanup(func() { editorWellKnown = orig })
}

func writeFakeEditor(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if runtime.GOOS == "windows" {
		path += ".cmd"
	}
	script := "#!/bin/sh\necho fake\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func editorPath(editors []Editor, kind EditorKind) string {
	for _, e := range editors {
		if e.Kind == kind {
			return e.Path
		}
	}
	return ""
}

func assertPaths(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("paths=%v; want %v", got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("paths=%v; want %v", got, want)
		}
	}
}
