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
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

// errReleaseNotFound means the GitHub release tag itself is missing (vsix 404
// and the release asset list 404). Other 404s, including checksums.txt, must
// not be treated as a missing tag.
var errReleaseNotFound = errors.New("github release not found")

type vscodeInstall struct {
	cliVersion string
	vscodePath string
	cursorPath string
	yes        bool

	run       execRunner
	github    *GitHub
	client    *http.Client
	compatURL string
	isTTY     func() bool
	skipExit  bool
}

func NewVscodeCommand(cliVersion string) *cobra.Command {
	return buildVscodeCommand(&vscodeInstall{cliVersion: cliVersion})
}

func buildVscodeCommand(v *vscodeInstall) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "vscode",
		Short:        "Manage the Versori VS Code and Cursor extension",
		SilenceUsage: true,
	}
	cmd.AddCommand(v.installCommand())
	return cmd
}

func (v *vscodeInstall) installCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the Versori extension into VS Code or Cursor",
		Long: `Install the Versori VS Code / Cursor extension.

Downloads the latest vsix compatible with this CLI and installs it with
the editor CLI (--install-extension --force).

With one resolved editor, installs immediately. If both VS Code and Cursor
are resolved, prompts you to choose where to install. Pass --yes to install
into every resolved editor; non-interactive shells with both editors require
--yes or --confirm.

Use --vscode-path and --cursor-path to point at editor CLIs that are not
on PATH.`,
		SilenceUsage: true,
		RunE:         v.runE,
	}

	f := cmd.Flags()
	f.StringVar(&v.vscodePath, "vscode-path", "", "Absolute path to the VS Code CLI (code)")
	f.StringVar(&v.cursorPath, "cursor-path", "", "Absolute path to the Cursor CLI (cursor)")
	flags.AddSkipPromptFlags(f, &v.yes)

	return cmd
}

func (v *vscodeInstall) runE(cmd *cobra.Command, _ []string) error {
	if err := v.install(cmd); err != nil {
		return v.fail(err)
	}
	return nil
}

func (v *vscodeInstall) install(cmd *cobra.Command) error {
	editors, err := ResolveEditors(v.vscodePath, v.cursorPath)
	if err != nil {
		return err
	}

	chosen, err := chooseEditors(editors, v.yes, v.stdinIsTTY())
	if err != nil {
		return err
	}
	if len(chosen) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Aborted; no changes were made.")
		return nil
	}

	table, err := FetchCompat(v.httpClient(), v.compatTableURL())
	if err != nil {
		return fmt.Errorf("failed to fetch the compatibility table: %w", err)
	}

	vsixVer, ok := PickVSIX(v.cliVersion, table)
	if !ok {
		return fmt.Errorf("%s", TooOldMessage())
	}

	dir, err := os.MkdirTemp("", "versori-vsix-*")
	if err != nil {
		return fmt.Errorf("failed to create a temp directory: %w", err)
	}
	defer os.RemoveAll(dir)

	vsixPath, err := downloadCompatibleVSIX(v.gitHub(), vsixVer, dir)
	if err != nil {
		return fmt.Errorf("failed to download the extension: %w", err)
	}

	for _, ed := range chosen {
		if err := InstallVSIX(ed.Path, vsixPath, v.run); err != nil {
			return fmt.Errorf("failed to install the extension into %s: %w", ed.Kind, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Installed the Versori extension into %s.\n", ed.Kind)
	}
	return nil
}

func (v *vscodeInstall) fail(err error) error {
	e := utils.NewExitError().WithMessage(err.Error())
	if v.skipExit {
		return e
	}
	e.Done()
	return e
}

func (v *vscodeInstall) stdinIsTTY() bool {
	if v.isTTY != nil {
		return v.isTTY()
	}
	return isatty.IsTerminal(os.Stdin.Fd())
}

func (v *vscodeInstall) httpClient() *http.Client {
	if v.client != nil {
		return v.client
	}
	return http.DefaultClient
}

func (v *vscodeInstall) compatTableURL() string {
	if v.compatURL != "" {
		return v.compatURL
	}
	return DefaultCompatURL
}

func (v *vscodeInstall) gitHub() *GitHub {
	if v.github != nil {
		return v.github
	}
	return &GitHub{}
}

func downloadCompatibleVSIX(g *GitHub, vsixVer, destDir string) (string, error) {
	repo := g.extRepo()
	stripped := strings.TrimPrefix(strings.TrimSpace(vsixVer), "v")
	if stripped == "" {
		return "", fmt.Errorf("empty vsix version")
	}
	want := "versori-vscode-" + stripped + ".vsix"

	var last error
	for _, tag := range []string{"v" + stripped, stripped} {
		path, err := downloadVSIXAtTag(g, repo, tag, want, destDir)
		if err == nil {
			return path, nil
		}
		last = err
		if !errors.Is(err, errReleaseNotFound) {
			return "", err
		}
	}
	if last != nil {
		return "", last
	}
	return "", fmt.Errorf("vsix asset not found for %s", vsixVer)
}

func downloadVSIXAtTag(g *GitHub, repo, tag, want, destDir string) (string, error) {
	vsixPath := filepath.Join(destDir, filepath.Base(want))
	err := g.DownloadReleaseFile(repo, tag, want, vsixPath)
	if isNotFound(err) {
		names, listErr := g.ReleaseAssetNames(repo, tag)
		if listErr != nil {
			if isNotFound(listErr) {
				return "", fmt.Errorf("%w: %w", errReleaseNotFound, listErr)
			}
			return "", listErr
		}
		picked, perr := PickVSIXAsset(names)
		if perr != nil {
			return "", perr
		}
		vsixPath = filepath.Join(destDir, filepath.Base(picked))
		if err = g.DownloadReleaseFile(repo, tag, picked, vsixPath); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	sumsPath := filepath.Join(destDir, "checksums.txt")
	if err := g.DownloadReleaseFile(repo, tag, "checksums.txt", sumsPath); err != nil {
		return "", fmt.Errorf("failed to download checksums.txt: %w", err)
	}
	if err := VerifySHA256(vsixPath, sumsPath, filepath.Base(vsixPath)); err != nil {
		return "", fmt.Errorf("failed to verify checksum: %w", err)
	}
	return vsixPath, nil
}

func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "status 404")
}
