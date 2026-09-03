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
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/utils"
)

const cliRepo = "versori/cli"

type updater struct {
	cliVersion string
	version    string
	vscodePath string
	cursorPath string

	run       execRunner
	github    *GitHub
	client    *http.Client
	compatURL string
	skipExit  bool

	goos         func() string
	executable   func() (string, error)
	sudoMove     func(tmp, dest string) error
	fetch        func() (CompatTable, error)
	pickVSIX     func(cliVersion string, table CompatTable) (string, bool)
	downloadVSIX func(g *GitHub, vsixVer, destDir string) (string, error)
}

func NewUpdateCommand(cliVersion string) *cobra.Command {
	return buildUpdateCommand(&updater{cliVersion: cliVersion})
}

func buildUpdateCommand(u *updater) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update this CLI binary and refresh an installed editor extension",
		Long: `Replace this versori binary with a GitHub release (latest, or --version).

The binary that is overwritten is the one you ran (after resolving symlinks),
not always /usr/local/bin/versori.

On Windows, download a release zip instead; the editor extension pass still runs.

If the Versori VS Code or Cursor extension is already installed, it is refreshed
to the latest vsix compatible with the target CLI. There is no prompt.

If replacing the binary requires elevated permissions, this command runs
sudo mv onto the resolved path (never sudo versori update).`,
		SilenceUsage: true,
		RunE:         u.runE,
	}

	f := cmd.Flags()
	f.StringVar(&u.version, "version", "", "CLI release tag to install (default: latest)")
	f.StringVar(&u.vscodePath, "vscode-path", "", "Absolute path to the VS Code CLI (code)")
	f.StringVar(&u.cursorPath, "cursor-path", "", "Absolute path to the Cursor CLI (cursor)")

	return cmd
}

func vsixCLIVersion(current, target string) string {
	return target
}

func (u *updater) runE(cmd *cobra.Command, _ []string) error {
	target, err := u.replaceBinary(cmd)
	if err != nil {
		return u.fail(err)
	}
	u.vsixPass(cmd, target)
	return nil
}

func (u *updater) replaceBinary(cmd *cobra.Command) (string, error) {
	target, err := u.targetVersion()
	if err != nil {
		return "", err
	}

	if u.osName() == "windows" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Windows is not supported for in-place binary update; download a release zip from https://github.com/versori/cli/releases")
		return target, nil
	}

	if versionsEqual(u.cliVersion, target) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "versori is already %s\n", target)
		return target, nil
	}

	dest, err := u.destPath()
	if err != nil {
		return "", fmt.Errorf("failed to resolve this executable: %w", err)
	}

	dir, err := os.MkdirTemp("", "versori-update-*")
	if err != nil {
		return "", fmt.Errorf("failed to create a temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	extracted, err := u.downloadCLI(target, dir)
	if err != nil {
		return "", err
	}

	if err := u.installBinary(extracted, dest); err != nil {
		return "", err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "updated versori to %s\n", target)
	return target, nil
}

func (u *updater) targetVersion() (string, error) {
	if v := strings.TrimSpace(u.version); v != "" {
		return v, nil
	}
	tag, err := u.gitHub().LatestTag(cliRepo)
	if err != nil {
		return "", fmt.Errorf("failed to resolve the latest CLI release: %w", err)
	}
	return tag, nil
}

func (u *updater) destPath() (string, error) {
	fn := os.Executable
	if u.executable != nil {
		fn = u.executable
	}
	exe, err := fn()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func (u *updater) downloadCLI(target, dir string) (string, error) {
	g := u.gitHub()
	name := CLITarballName(target, runtime.GOOS, runtime.GOARCH)
	tarball := filepath.Join(dir, name)
	sums := filepath.Join(dir, "checksums.txt")

	if err := g.DownloadReleaseFile(cliRepo, target, name, tarball); err != nil {
		return "", fmt.Errorf("failed to download %s: %w", name, err)
	}
	if err := g.DownloadReleaseFile(cliRepo, target, "checksums.txt", sums); err != nil {
		return "", fmt.Errorf("failed to download checksums.txt: %w", err)
	}
	if err := VerifySHA256(tarball, sums, name); err != nil {
		return "", fmt.Errorf("failed to verify checksum: %w", err)
	}

	extracted, err := ExtractVersoriBinary(tarball, dir)
	if err != nil {
		return "", fmt.Errorf("failed to extract the CLI: %w", err)
	}
	if err := os.Chmod(extracted, 0o755); err != nil {
		return "", fmt.Errorf("failed to chmod the extracted binary: %w", err)
	}
	return extracted, nil
}

func (u *updater) installBinary(extracted, dest string) error {
	tmp := dest + ".new"
	if err := copyFile(extracted, tmp); err != nil {
		if !isPermissionErr(err) {
			return fmt.Errorf("failed to write %s: %w", tmp, err)
		}
		if err := u.moveWithSudo(extracted, dest); err != nil {
			return replaceFail(dest, err)
		}
		return nil
	}

	if err := os.Rename(tmp, dest); err != nil {
		if !isPermissionErr(err) {
			_ = os.Remove(tmp)
			return fmt.Errorf("failed to replace %s: %w", dest, err)
		}
		if err := u.moveWithSudo(tmp, dest); err != nil {
			_ = os.Remove(tmp)
			return replaceFail(dest, err)
		}
		return nil
	}
	return nil
}

func replaceFail(dest string, err error) error {
	return fmt.Errorf("failed to replace %s: %v\n%s", dest, err, curlInstall)
}

func (u *updater) moveWithSudo(tmp, dest string) error {
	if u.sudoMove != nil {
		return u.sudoMove(tmp, dest)
	}
	c := exec.Command("sudo", "mv", tmp, dest)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func (u *updater) vsixPass(cmd *cobra.Command, target string) {
	errOut := cmd.ErrOrStderr()
	out := cmd.OutOrStdout()

	editors := u.resolveEditors(errOut)
	if len(editors) == 0 {
		_, _ = fmt.Fprintln(out, "No VS Code or Cursor CLI was found.")
		return
	}

	var need []Editor
	for _, ed := range editors {
		has, err := HasExtension(ed.Path, ExtensionID, u.run)
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "warning: failed to list extensions for %s: %v\n", ed.Kind, err)
			continue
		}
		if has {
			need = append(need, ed)
		}
	}
	if len(need) == 0 {
		return
	}

	table, err := u.fetchCompat()
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "warning: failed to fetch the compatibility table: %v\n", err)
		return
	}

	vsixVer, ok := u.pick(vsixCLIVersion(u.cliVersion, target), table)
	if !ok {
		_, _ = fmt.Fprintf(errOut, "warning: %s\n", TooOldMessage())
		return
	}

	dir, err := os.MkdirTemp("", "versori-vsix-*")
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "warning: failed to create a temp directory: %v\n", err)
		return
	}
	defer func() { _ = os.RemoveAll(dir) }()

	vsixPath, err := u.downloadExt(vsixVer, dir)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "warning: failed to download the extension: %v\n", err)
		return
	}

	for _, ed := range need {
		if err := InstallVSIX(ed.Path, vsixPath, u.run); err != nil {
			_, _ = fmt.Fprintf(errOut, "warning: failed to install the extension into %s: %v\n", ed.Kind, err)
			continue
		}
		_, _ = fmt.Fprintf(out, "Updated the Versori extension in %s.\n", ed.Kind)
	}
}

// resolveEditors uses LookEditor per editor so a bad --vscode-path / --cursor-path
// warns and skips that editor instead of aborting the way ResolveEditors does.
func (u *updater) resolveEditors(errOut io.Writer) []Editor {
	var editors []Editor
	for _, item := range []struct {
		kind EditorKind
		flag string
	}{
		{EditorVSCode, u.vscodePath},
		{EditorCursor, u.cursorPath},
	} {
		path, _, err := LookEditor(item.kind, item.flag)
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "warning: %v\n", err)
			continue
		}
		if path == "" {
			continue
		}
		editors = append(editors, Editor{Kind: item.kind, Path: path})
	}
	return editors
}

func (u *updater) fail(err error) error {
	e := utils.NewExitError().WithMessage(err.Error())
	if u.skipExit {
		return e
	}
	e.Done()
	return e
}

func (u *updater) osName() string {
	if u.goos != nil {
		return u.goos()
	}
	return runtime.GOOS
}

func (u *updater) httpClient() *http.Client {
	if u.client != nil {
		return u.client
	}
	return http.DefaultClient
}

func (u *updater) compatTableURL() string {
	if u.compatURL != "" {
		return u.compatURL
	}
	return DefaultCompatURL
}

func (u *updater) gitHub() *GitHub {
	if u.github != nil {
		return u.github
	}
	return &GitHub{}
}

func (u *updater) fetchCompat() (CompatTable, error) {
	if u.fetch != nil {
		return u.fetch()
	}
	return FetchCompat(u.httpClient(), u.compatTableURL())
}

func (u *updater) pick(cliVersion string, table CompatTable) (string, bool) {
	if u.pickVSIX != nil {
		return u.pickVSIX(cliVersion, table)
	}
	return PickVSIX(cliVersion, table)
}

func (u *updater) downloadExt(vsixVer, destDir string) (string, error) {
	if u.downloadVSIX != nil {
		return u.downloadVSIX(u.gitHub(), vsixVer, destDir)
	}
	return downloadCompatibleVSIX(u.gitHub(), vsixVer, destDir)
}

func versionsEqual(a, b string) bool {
	return stripV(a) == stripV(b)
}

func stripV(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dest)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dest)
		return closeErr
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		_ = os.Remove(dest)
		return err
	}
	return nil
}

func isPermissionErr(err error) bool {
	if err == nil {
		return false
	}
	if os.IsPermission(err) || errors.Is(err, os.ErrPermission) {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == syscall.EPERM || errno == syscall.EACCES
	}
	return false
}
