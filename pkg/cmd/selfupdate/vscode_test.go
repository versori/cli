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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestChooseEditors(t *testing.T) {
	vscode := Editor{Kind: EditorVSCode, Path: "/bin/code"}
	cursor := Editor{Kind: EditorCursor, Path: "/bin/cursor"}

	t.Run("none", func(t *testing.T) {
		_, err := chooseEditors(nil, true, false)
		if !errors.Is(err, errNoEditors) {
			t.Fatalf("err=%v; want errNoEditors", err)
		}
		if !strings.Contains(err.Error(), "--vscode-path") || !strings.Contains(err.Error(), "--cursor-path") {
			t.Fatalf("err=%v; want path flags", err)
		}
	})

	t.Run("yes all", func(t *testing.T) {
		got, err := chooseEditors([]Editor{vscode, cursor}, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0].Kind != EditorVSCode || got[1].Kind != EditorCursor {
			t.Fatalf("got=%v", got)
		}
	})

	t.Run("single non-tty installs without yes", func(t *testing.T) {
		got, err := chooseEditors([]Editor{vscode}, false, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Kind != EditorVSCode {
			t.Fatalf("got=%v", got)
		}
	})

	t.Run("both non-tty without yes", func(t *testing.T) {
		_, err := chooseEditors([]Editor{vscode, cursor}, false, false)
		if !errors.Is(err, errNeedTTY) {
			t.Fatalf("err=%v; want errNeedTTY", err)
		}
	})
}

func TestVscodeInstallBothEditorsNonTTYWithoutYesDoesNotRun(t *testing.T) {
	home := isolateCmdEnv(t)
	dir := t.TempDir()
	writeFakeEditor(t, dir, "code")
	writeFakeEditor(t, dir, "cursor")
	t.Setenv("PATH", dir)

	var ran bool
	cmd := testVscodeCmd(t, &vscodeInstall{
		cliVersion: "0.1.0",
		run: func(name string, args ...string) ([]byte, error) {
			ran = true
			return nil, nil
		},
	})
	cmd.SetArgs([]string{"install"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("want error")
	}
	if !strings.Contains(err.Error(), "TTY") && !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err=%v", err)
	}
	if ran {
		t.Fatal("run must not be called without --yes on a non-TTY")
	}
	assertNoConfigYAML(t, home)
}

func TestVscodeInstallSingleEditorInstallsWithoutPrompt(t *testing.T) {
	home := isolateCmdEnv(t)
	dir := t.TempDir()
	codePath := writeFakeEditor(t, dir, "code")
	t.Setenv("PATH", dir)

	vsixBody := []byte("fake-vsix")
	sum := sha256.Sum256(vsixBody)
	hash := hex.EncodeToString(sum[:])
	const vsixName = "versori-vscode-0.1.0.vsix"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compat.json":
			_, _ = w.Write([]byte(`{"extension_id":"versori.versori-vscode","releases":[{"vsix":"0.1.0","min_cli":"0.1.0"}]}`))
		case "/repos/versori/versori-vscode-extension/releases/tags/v0.1.0":
			_, _ = w.Write([]byte(`{"assets":[{"name":"versori-vscode-0.1.0.vsix"},{"name":"checksums.txt"}]}`))
		case "/versori/versori-vscode-extension/releases/download/v0.1.0/" + vsixName:
			_, _ = w.Write(vsixBody)
		case "/versori/versori-vscode-extension/releases/download/v0.1.0/checksums.txt":
			_, _ = w.Write([]byte(hash + "  " + vsixName + "\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	var calls [][]string
	var bins []string
	cmd := testVscodeCmd(t, &vscodeInstall{
		cliVersion: "0.1.0",
		client:     srv.Client(),
		compatURL:  srv.URL + "/compat.json",
		github: &GitHub{
			Client:       srv.Client(),
			ExtRepo:      "versori/versori-vscode-extension",
			apiBase:      srv.URL,
			downloadBase: srv.URL,
		},
		run: func(name string, args ...string) ([]byte, error) {
			bins = append(bins, name)
			calls = append(calls, append([]string{}, args...))
			return nil, nil
		},
	})
	cmd.SetArgs([]string{"install"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("run calls=%v", calls)
	}
	if bins[0] != codePath {
		t.Fatalf("bin=%q; want %q", bins[0], codePath)
	}
	if len(calls[0]) != 3 || calls[0][0] != "--install-extension" || calls[0][2] != "--force" {
		t.Fatalf("args=%v", calls[0])
	}
	if !strings.HasSuffix(calls[0][1], ".vsix") {
		t.Fatalf("vsix path=%q", calls[0][1])
	}
	assertNoConfigYAML(t, home)
}

func TestVscodeInstallChecksums404DoesNotRetryBareTag(t *testing.T) {
	home := isolateCmdEnv(t)
	dir := t.TempDir()
	writeFakeEditor(t, dir, "code")
	t.Setenv("PATH", dir)

	vsixBody := []byte("fake-vsix")
	sum := sha256.Sum256(vsixBody)
	hash := hex.EncodeToString(sum[:])
	const vsixName = "versori-vscode-0.1.0.vsix"
	var bareHits int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compat.json":
			_, _ = w.Write([]byte(`{"extension_id":"versori.versori-vscode","releases":[{"vsix":"0.1.0","min_cli":"0.1.0"}]}`))
		case "/repos/versori/versori-vscode-extension/releases/tags/v0.1.0":
			_, _ = w.Write([]byte(`{"assets":[{"name":"versori-vscode-0.1.0.vsix"}]}`))
		case "/versori/versori-vscode-extension/releases/download/v0.1.0/" + vsixName:
			_, _ = w.Write(vsixBody)
		case "/versori/versori-vscode-extension/releases/download/v0.1.0/checksums.txt":
			w.WriteHeader(http.StatusNotFound)
		case "/repos/versori/versori-vscode-extension/releases/tags/0.1.0",
			"/versori/versori-vscode-extension/releases/download/0.1.0/" + vsixName,
			"/versori/versori-vscode-extension/releases/download/0.1.0/checksums.txt":
			bareHits++
			// A retry would succeed if we served a complete bare-tag release.
			if strings.HasSuffix(r.URL.Path, vsixName) {
				_, _ = w.Write(vsixBody)
				return
			}
			if strings.HasSuffix(r.URL.Path, "checksums.txt") {
				_, _ = w.Write([]byte(hash + "  " + vsixName + "\n"))
				return
			}
			_, _ = w.Write([]byte(`{"assets":[{"name":"versori-vscode-0.1.0.vsix"},{"name":"checksums.txt"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	var ran bool
	cmd := testVscodeCmd(t, &vscodeInstall{
		cliVersion: "0.1.0",
		client:     srv.Client(),
		compatURL:  srv.URL + "/compat.json",
		github: &GitHub{
			Client:       srv.Client(),
			ExtRepo:      "versori/versori-vscode-extension",
			apiBase:      srv.URL,
			downloadBase: srv.URL,
		},
		run: func(name string, args ...string) ([]byte, error) {
			ran = true
			return nil, nil
		},
	})
	cmd.SetArgs([]string{"install", "--yes"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("want error when checksums.txt is 404")
	}
	if ran {
		t.Fatal("must not install when checksums.txt is missing")
	}
	if bareHits != 0 {
		t.Fatalf("bare tag requests=%d; want 0 (checksum 404 must not retry the other tag)", bareHits)
	}
	assertNoConfigYAML(t, home)
}

func TestVscodeInstallDoesNotCreateConfigYAML(t *testing.T) {
	home := isolateCmdEnv(t)
	cmd := testVscodeCmd(t, &vscodeInstall{
		cliVersion: "0.1.0",
		run: func(name string, args ...string) ([]byte, error) {
			t.Fatal("run must not be called")
			return nil, nil
		},
	})
	cmd.SetArgs([]string{"install"})
	_ = cmd.Execute()
	assertNoConfigYAML(t, home)
}

func TestVscodeInstallNoForceFlag(t *testing.T) {
	cmd := NewVscodeCommand("0.1.0")
	install := findSub(cmd, "install")
	if install == nil {
		t.Fatal("missing install subcommand")
	}
	if install.Flags().Lookup("force") != nil {
		t.Fatal("unexpected --force flag")
	}
	if install.Flags().Lookup("yes") == nil || install.Flags().Lookup("confirm") == nil {
		t.Fatal("missing skip-prompt flags")
	}
	if install.Flags().Lookup("vscode-path") == nil || install.Flags().Lookup("cursor-path") == nil {
		t.Fatal("missing editor path flags")
	}
	if install.PersistentPreRun != nil || cmd.PersistentPreRun != nil {
		t.Fatal("vscode commands must not load config")
	}
}

func testVscodeCmd(t *testing.T, v *vscodeInstall) *cobra.Command {
	t.Helper()
	v.skipExit = true
	if v.isTTY == nil {
		v.isTTY = func() bool { return false }
	}
	if v.run == nil {
		v.run = func(name string, args ...string) ([]byte, error) {
			t.Fatalf("unexpected run %s %v", name, args)
			return nil, nil
		}
	}
	cmd := buildVscodeCommand(v)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd
}

func isolateCmdEnv(t *testing.T) string {
	t.Helper()
	isolateWellKnown(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	return home
}

func assertNoConfigYAML(t *testing.T, home string) {
	t.Helper()
	p := filepath.Join(home, ".versori", "config.yaml")
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("created %s: %v", p, err)
	}
}

func findSub(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
