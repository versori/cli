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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVsixCLIVersionUsesTarget(t *testing.T) {
	if got := vsixCLIVersion("0.0.1", "v0.0.9"); got != "v0.0.9" {
		t.Fatalf("vsixCLIVersion=%q; want target v0.0.9", got)
	}
	if got := vsixCLIVersion("v0.0.9", "v1.2.3"); got != "v1.2.3" {
		t.Fatalf("vsixCLIVersion=%q; want target v1.2.3", got)
	}
}

func TestUpdateVsixPassUsesTargetNotCurrent(t *testing.T) {
	home := isolateCmdEnv(t)
	dir := t.TempDir()
	writeFakeEditor(t, dir, "code")
	t.Setenv("PATH", dir)

	var picked string
	var downloaded bool
	cmd := testUpdateCmd(t, &updater{
		cliVersion: "0.0.1",
		goos:       func() string { return "windows" },
		fetch: func() (CompatTable, error) {
			return CompatTable{
				ExtensionID: ExtensionID,
				Releases:    []CompatRelease{{Vsix: "0.2.0", MinCLI: "0.0.9"}},
			}, nil
		},
		pickVSIX: func(cliVersion string, table CompatTable) (string, bool) {
			picked = cliVersion
			return "0.2.0", true
		},
		downloadVSIX: func(g *GitHub, vsixVer, destDir string) (string, error) {
			downloaded = true
			p := filepath.Join(destDir, "ext.vsix")
			if err := os.WriteFile(p, []byte("vsix"), 0o644); err != nil {
				return "", err
			}
			return p, nil
		},
		run: func(name string, args ...string) ([]byte, error) {
			if len(args) > 0 && args[0] == "--list-extensions" {
				return []byte(ExtensionID + "\n"), nil
			}
			return nil, nil
		},
	})
	cmd.SetArgs([]string{"--version", "v0.0.9"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if picked != "v0.0.9" {
		t.Fatalf("PickVSIX version=%q; want target v0.0.9 (not current 0.0.1)", picked)
	}
	if !downloaded {
		t.Fatal("expected stub downloadVSIX to run")
	}
	assertNoConfigYAML(t, home)
}

func TestUpdateDoesNotCreateConfigYAML(t *testing.T) {
	home := isolateCmdEnv(t)
	cmd := testUpdateCmd(t, &updater{
		cliVersion: "0.1.0",
		goos:       func() string { return "windows" },
	})
	cmd.SetArgs([]string{"--version", "0.1.0"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	assertNoConfigYAML(t, home)
}

func TestUpdateNoForceFlag(t *testing.T) {
	cmd := NewUpdateCommand("0.1.0")
	if cmd.Flags().Lookup("force") != nil {
		t.Fatal("unexpected --force flag")
	}
	if cmd.Flags().Lookup("yes") != nil {
		t.Fatal("unexpected --yes flag")
	}
	if cmd.Flags().Lookup("version") == nil {
		t.Fatal("missing --version")
	}
	if cmd.Flags().Lookup("vscode-path") == nil || cmd.Flags().Lookup("cursor-path") == nil {
		t.Fatal("missing editor path flags")
	}
	if cmd.PersistentPreRun != nil {
		t.Fatal("update must not load config")
	}
}

func TestUpdateBadEditorFlagWarnsAndSucceeds(t *testing.T) {
	home := isolateCmdEnv(t)
	var stderr bytes.Buffer
	cmd := testUpdateCmd(t, &updater{
		cliVersion: "0.1.0",
		goos:       func() string { return "windows" },
	})
	cmd.SetErr(&stderr)
	missing := filepath.Join(t.TempDir(), "no-such-code")
	cmd.SetArgs([]string{"--version", "0.1.0", "--vscode-path", missing})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "warning:") {
		t.Fatalf("stderr=%q; want warning", stderr.String())
	}
	assertNoConfigYAML(t, home)
}

func TestUpdateCompatFetchFailWarns(t *testing.T) {
	home := isolateCmdEnv(t)
	dir := t.TempDir()
	writeFakeEditor(t, dir, "code")
	t.Setenv("PATH", dir)

	var stderr bytes.Buffer
	var picked bool
	cmd := testUpdateCmd(t, &updater{
		cliVersion: "0.1.0",
		goos:       func() string { return "windows" },
		fetch: func() (CompatTable, error) {
			return CompatTable{}, errors.New("network down")
		},
		pickVSIX: func(cliVersion string, table CompatTable) (string, bool) {
			picked = true
			return "", false
		},
		run: func(name string, args ...string) ([]byte, error) {
			if len(args) > 0 && args[0] == "--list-extensions" {
				return []byte(ExtensionID + "\n"), nil
			}
			t.Fatalf("unexpected run %s %v", name, args)
			return nil, nil
		},
	})
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--version", "0.1.0"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if picked {
		t.Fatal("PickVSIX must not run when compat fetch fails")
	}
	if !strings.Contains(stderr.String(), "warning:") || !strings.Contains(stderr.String(), "compatibility table") {
		t.Fatalf("stderr=%q", stderr.String())
	}
	assertNoConfigYAML(t, home)
}

func TestUpdateSkipsVsixWhenExtensionMissing(t *testing.T) {
	home := isolateCmdEnv(t)
	dir := t.TempDir()
	writeFakeEditor(t, dir, "code")
	t.Setenv("PATH", dir)

	var picked bool
	cmd := testUpdateCmd(t, &updater{
		cliVersion: "0.1.0",
		goos:       func() string { return "windows" },
		fetch: func() (CompatTable, error) {
			t.Fatal("must not fetch compat when no editor has the extension")
			return CompatTable{}, nil
		},
		pickVSIX: func(cliVersion string, table CompatTable) (string, bool) {
			picked = true
			return "", false
		},
		run: func(name string, args ...string) ([]byte, error) {
			if len(args) > 0 && args[0] == "--list-extensions" {
				return []byte("other.publisher.ext\n"), nil
			}
			t.Fatalf("unexpected run %s %v", name, args)
			return nil, nil
		},
	})
	cmd.SetArgs([]string{"--version", "0.1.0"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if picked {
		t.Fatal("PickVSIX must not run when the extension is not installed")
	}
	assertNoConfigYAML(t, home)
}

func TestUpdateReplacesWritableBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("binary replace is not attempted on Windows")
	}

	home := isolateCmdEnv(t)
	destDir := t.TempDir()
	dest := filepath.Join(destDir, "versori")
	if err := os.WriteFile(dest, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	payload := []byte("new-binary")
	tarballName := CLITarballName("v0.0.9", runtime.GOOS, runtime.GOARCH)
	tarPath := filepath.Join(t.TempDir(), tarballName)
	writeTarGz(t, tarPath, map[string][]byte{"versori": payload})
	tarBytes, err := os.ReadFile(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(tarBytes)
	hash := hex.EncodeToString(sum[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versori/cli/releases/download/v0.0.9/" + tarballName:
			_, _ = w.Write(tarBytes)
		case "/versori/cli/releases/download/v0.0.9/checksums.txt":
			_, _ = w.Write([]byte(hash + "  " + tarballName + "\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	cmd := testUpdateCmd(t, &updater{
		cliVersion: "0.0.1",
		executable: func() (string, error) { return dest, nil },
		github: &GitHub{
			Client:       srv.Client(),
			downloadBase: srv.URL,
		},
	})
	cmd.SetArgs([]string{"--version", "v0.0.9"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("dest=%q; want new-binary", got)
	}
	if _, err := os.Stat(dest + ".new"); !os.IsNotExist(err) {
		t.Fatalf("left behind %s.new: %v", dest, err)
	}
	assertNoConfigYAML(t, home)
}

func testUpdateCmd(t *testing.T, u *updater) *cobra.Command {
	t.Helper()
	u.skipExit = true
	if u.fetch == nil {
		u.fetch = func() (CompatTable, error) {
			return CompatTable{}, nil
		}
	}
	if u.downloadVSIX == nil {
		u.downloadVSIX = func(g *GitHub, vsixVer, destDir string) (string, error) {
			t.Fatal("downloadVSIX must be stubbed; refusing live GitHub")
			return "", nil
		}
	}
	if u.github == nil {
		u.github = &GitHub{
			apiBase:      "http://127.0.0.1:1",
			downloadBase: "http://127.0.0.1:1",
		}
	}
	if u.run == nil {
		u.run = func(name string, args ...string) ([]byte, error) {
			t.Fatalf("unexpected run %s %v", name, args)
			return nil, nil
		}
	}
	cmd := buildUpdateCommand(u)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd
}
