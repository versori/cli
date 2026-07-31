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

package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	updateRepo       = "versori/cli"
	updateBinaryName = "versori"
	// maxArchiveSize caps how much we will read out of the release archive, so a
	// malformed or hostile archive can't exhaust local disk.
	maxArchiveSize = 256 << 20 // 256 MiB
)

type updateOptions struct {
	tag     string
	check   bool
	force   bool
	timeout time.Duration
}

func newUpdateCommand() *cobra.Command {
	o := &updateOptions{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the versori CLI to the latest release",
		Long: `Update downloads the latest versori release from GitHub and replaces the
currently running binary in place.

The download is verified against the checksums.txt published with the release
before anything is written. Use --check to see whether an update is available
without installing it.

If the binary lives somewhere you cannot write to (for example /usr/local/bin),
re-run this command with sudo.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd)
		},
	}

	f := cmd.Flags()
	f.StringVar(&o.tag, "tag", "", "Install a specific release tag (for example v0.2.0) instead of the latest")
	f.BoolVar(&o.check, "check", false, "Only report whether a newer version is available; don't install it")
	f.BoolVar(&o.force, "force", false, "Reinstall even if the target version is already installed")
	f.DurationVar(&o.timeout, "timeout", 5*time.Minute, "Timeout for the download")

	return cmd
}

func (o *updateOptions) run(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	client := &http.Client{Timeout: o.timeout}

	target := o.tag
	if target == "" {
		_, _ = fmt.Fprintln(out, "Checking for updates...")

		latest, err := latestReleaseTag(cmd.Context(), client)
		if err != nil {
			return err
		}

		target = latest
	}

	current := normaliseVersion(version)
	wanted := normaliseVersion(target)

	_, _ = fmt.Fprintf(out, "Installed: %s\nAvailable: %s\n", current, wanted)

	switch {
	case o.check:
		if compareVersions(wanted, current) > 0 {
			_, _ = fmt.Fprintf(out, "\nA newer version is available. Run 'versori update' to install it.\n")
		} else {
			_, _ = fmt.Fprintln(out, "\nYou're up to date.")
		}

		return nil
	case wanted == current && !o.force:
		_, _ = fmt.Fprintln(out, "\nAlready up to date. Pass --force to reinstall.")

		return nil
	case compareVersions(wanted, current) < 0 && !o.force:
		return fmt.Errorf("%s is older than the installed version %s: pass --force to downgrade", wanted, current)
	}

	dest, err := executablePath()
	if err != nil {
		return err
	}

	// Fail early with a clear message rather than after a long download.
	if err := checkWritable(filepath.Dir(dest)); err != nil {
		return err
	}

	assetName := releaseAssetName(wanted)
	baseURL := fmt.Sprintf("https://github.com/%s/releases/download/v%s", updateRepo, wanted)

	_, _ = fmt.Fprintf(out, "\nDownloading %s...\n", assetName)

	archive, err := download(cmd.Context(), client, baseURL+"/"+assetName)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(out, "Verifying checksum...")

	checksums, err := download(cmd.Context(), client, baseURL+"/checksums.txt")
	if err != nil {
		return fmt.Errorf("failed to download checksums.txt: %w", err)
	}

	if err := verifyChecksum(archive, checksums, assetName); err != nil {
		return err
	}

	binary, err := extractBinary(archive, assetName)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "Installing to %s...\n", dest)

	if err := replaceExecutable(dest, binary); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "\nUpdated versori %s -> %s\n", current, wanted)

	return nil
}

// latestReleaseTag returns the newest published release tag, without the leading "v".
func latestReleaseTag(ctx context.Context, client *http.Client) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", updateRepo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query the latest release: %w", err)
	}
	defer res.Body.Close() //nolint:errcheck

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to query the latest release: GitHub returned %s", res.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to parse the latest release: %w", err)
	}

	if payload.TagName == "" {
		return "", errors.New("the latest release has no tag name")
	}

	return payload.TagName, nil
}

// releaseAssetName builds the goreleaser archive name for the running platform.
// It must stay in sync with the name_template in .goreleaser.yaml.
func releaseAssetName(version string) string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("cli_%s_%s_%s.%s", version, runtime.GOOS, runtime.GOARCH, ext)
}

func download(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer res.Body.Close() //nolint:errcheck

	if res.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s was not found: check the release exists and publishes an asset for %s/%s",
			url, runtime.GOOS, runtime.GOARCH)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download %s: server returned %s", url, res.Status)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, maxArchiveSize))
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", url, err)
	}

	return body, nil
}

// verifyChecksum checks archive against the entry for assetName in a
// goreleaser-style checksums.txt ("<sha256>  <filename>" per line).
func verifyChecksum(archive, checksums []byte, assetName string) error {
	var expected string

	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			expected = fields[0]

			break
		}
	}

	if expected == "" {
		return fmt.Errorf("no checksum published for %s", assetName)
	}

	sum := sha256.Sum256(archive)

	if actual := hex.EncodeToString(sum[:]); actual != expected {
		return fmt.Errorf("checksum mismatch for %s:\n  expected: %s\n  actual:   %s\n"+
			"the download may be corrupted or tampered with", assetName, expected, actual)
	}

	return nil
}

// extractBinary pulls the versori executable out of a release archive.
func extractBinary(archive []byte, assetName string) ([]byte, error) {
	name := updateBinaryName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	if strings.HasSuffix(assetName, ".zip") {
		return extractFromZip(archive, name)
	}

	return extractFromTarGz(archive, name)
}

func extractFromTarGz(archive []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("failed to read the release archive: %w", err)
	}
	defer gz.Close() //nolint:errcheck

	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to read the release archive: %w", err)
		}

		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != name {
			continue
		}

		binary, err := io.ReadAll(io.LimitReader(tr, maxArchiveSize))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s from the release archive: %w", name, err)
		}

		return binary, nil
	}

	return nil, fmt.Errorf("%s was not found in the release archive", name)
}

func extractFromZip(archive []byte, name string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("failed to read the release archive: %w", err)
	}

	for _, file := range zr.File {
		if file.FileInfo().IsDir() || filepath.Base(file.Name) != name {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to read %s from the release archive: %w", name, err)
		}
		defer rc.Close() //nolint:errcheck

		binary, err := io.ReadAll(io.LimitReader(rc, maxArchiveSize))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s from the release archive: %w", name, err)
		}

		return binary, nil
	}

	return nil, fmt.Errorf("%s was not found in the release archive", name)
}

// executablePath returns the fully resolved path of the running binary, so we
// replace the real file rather than a symlink pointing at it.
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to locate the running binary: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s: %w", exe, err)
	}

	return resolved, nil
}

// checkWritable verifies we can create files in dir before starting a download.
func checkWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".versori-update-*")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("cannot write to %s: re-run with 'sudo versori update'", dir)
		}

		return fmt.Errorf("cannot write to %s: %w", dir, err)
	}

	name := f.Name()

	_ = f.Close()
	_ = os.Remove(name)

	return nil
}

// replaceExecutable atomically swaps the binary at dest for the new one. The new
// file is staged in the same directory so the rename stays on one filesystem.
func replaceExecutable(dest string, binary []byte) error {
	dir := filepath.Dir(dest)

	staged, err := os.CreateTemp(dir, ".versori-update-*")
	if err != nil {
		return fmt.Errorf("failed to stage the new binary in %s: %w", dir, err)
	}

	stagedName := staged.Name()

	cleanup := func() {
		_ = staged.Close()
		_ = os.Remove(stagedName)
	}

	if _, err := staged.Write(binary); err != nil {
		cleanup()

		return fmt.Errorf("failed to write the new binary: %w", err)
	}

	if err := staged.Close(); err != nil {
		_ = os.Remove(stagedName)

		return fmt.Errorf("failed to write the new binary: %w", err)
	}

	// Preserve the existing mode where we can, otherwise fall back to 0755.
	mode := os.FileMode(0o755)
	if info, err := os.Stat(dest); err == nil {
		mode = info.Mode().Perm()
	}

	if err := os.Chmod(stagedName, mode); err != nil {
		_ = os.Remove(stagedName)

		return fmt.Errorf("failed to set permissions on the new binary: %w", err)
	}

	// Windows won't let us rename over a running executable, so move the current
	// one aside first and keep it around for the OS to release.
	var backup string

	if runtime.GOOS == "windows" {
		backup = dest + ".old"

		_ = os.Remove(backup)

		if err := os.Rename(dest, backup); err != nil {
			_ = os.Remove(stagedName)

			return fmt.Errorf("failed to move the current binary aside: %w", err)
		}
	}

	if err := os.Rename(stagedName, dest); err != nil {
		_ = os.Remove(stagedName)

		if backup != "" {
			_ = os.Rename(backup, dest)
		}

		return fmt.Errorf("failed to install the new binary to %s: %w", dest, err)
	}

	if backup != "" {
		// Best effort: this fails while the old image is still mapped, and the
		// next update will clear it.
		_ = os.Remove(backup)
	}

	return nil
}

// normaliseVersion strips a leading "v" and any surrounding whitespace so tags
// and ldflags-injected versions compare consistently.
func normaliseVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// compareVersions compares dotted numeric versions, returning -1, 0 or 1.
// Non-numeric segments (pre-release suffixes, "dev" builds) compare as strings,
// which is enough to decide whether an update is worth offering.
func compareVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")

	for i := 0; i < len(as) || i < len(bs); i++ {
		var av, bv string

		if i < len(as) {
			av = as[i]
		}

		if i < len(bs) {
			bv = bs[i]
		}

		an, aerr := strconv.Atoi(av)
		bn, berr := strconv.Atoi(bv)

		if aerr == nil && berr == nil {
			if an != bn {
				if an < bn {
					return -1
				}

				return 1
			}

			continue
		}

		if av != bv {
			if av < bv {
				return -1
			}

			return 1
		}
	}

	return 0
}
