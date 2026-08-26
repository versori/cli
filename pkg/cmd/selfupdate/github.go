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
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultCLIRepo = "versori/cli"
	defaultExtRepo = "versori/versori-vscode-extension"
	githubAPIBase  = "https://api.github.com"
	githubDLBase   = "https://github.com"
)

type GitHub struct {
	Client       *http.Client
	CLIRepo      string
	ExtRepo      string
	apiBase      string
	downloadBase string
}

func (g *GitHub) LatestTag(repo string) (string, error) {
	if repo == "" {
		repo = g.cliRepo()
	}
	url := g.apiBaseURL() + "/repos/" + repo + "/releases/latest"
	resp, err := g.http().Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github latest: unexpected status %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("github latest: empty tag_name")
	}
	return payload.TagName, nil
}

func (g *GitHub) DownloadReleaseFile(repo, tag, name, dest string) error {
	if repo == "" {
		repo = g.cliRepo()
	}
	url := g.downloadBaseURL() + "/" + repo + "/releases/download/" + tag + "/" + name
	resp, err := g.http().Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("github download %s: unexpected status %d", name, resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(dest)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dest)
		return closeErr
	}
	return nil
}

func VerifySHA256(path, checksumsPath, filename string) error {
	sums, err := os.ReadFile(checksumsPath)
	if err != nil {
		return err
	}

	var expected string
	for _, line := range strings.Split(string(sums), "\n") {
		if !strings.Contains(line, filename) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		expected = fields[0]
		break
	}
	if expected == "" {
		return fmt.Errorf("checksum for %s not found", filename)
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", filename, expected, actual)
	}
	return nil
}

func CLITarballName(version, goos, goarch string) string {
	stripped := strings.TrimPrefix(version, "v")
	osName := goos
	arch := goarch
	switch osName {
	case "Darwin", "darwin":
		osName = "darwin"
	case "Linux", "linux":
		osName = "linux"
	}
	switch arch {
	case "x86_64", "amd64":
		arch = "amd64"
	case "arm64", "aarch64":
		arch = "arm64"
	}
	return fmt.Sprintf("cli_%s_%s_%s.tar.gz", stripped, osName, arch)
}

func ExtractVersoriBinary(tarball, destDir string) (string, error) {
	f, err := os.Open(tarball)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		rel := filepath.Clean(hdr.Name)
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
			return "", fmt.Errorf("refusing to extract %q", hdr.Name)
		}

		target := filepath.Join(destDir, rel)
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) && target != filepath.Clean(destDir) {
			return "", fmt.Errorf("refusing to extract %q", hdr.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
		if err != nil {
			return "", err
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	}

	binaryPath := filepath.Join(destDir, "versori")
	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("binary 'versori' not found in the archive")
	}
	return binaryPath, nil
}

func PickVSIXAsset(names []string) (string, error) {
	var preferred, vsix []string
	for _, name := range names {
		base := filepath.Base(name)
		if !strings.HasSuffix(strings.ToLower(base), ".vsix") {
			continue
		}
		vsix = append(vsix, name)
		ok, err := filepath.Match("versori-vscode-*.vsix", base)
		if err == nil && ok {
			preferred = append(preferred, name)
		}
	}
	if len(preferred) > 0 {
		return preferred[0], nil
	}
	if len(vsix) == 1 {
		return vsix[0], nil
	}
	if len(vsix) == 0 {
		return "", fmt.Errorf("no .vsix asset")
	}
	return "", fmt.Errorf("multiple .vsix assets")
}

func (g *GitHub) http() *http.Client {
	if g != nil && g.Client != nil {
		return g.Client
	}
	return http.DefaultClient
}

func (g *GitHub) cliRepo() string {
	if g != nil && g.CLIRepo != "" {
		return g.CLIRepo
	}
	return defaultCLIRepo
}

func (g *GitHub) extRepo() string {
	if g != nil && g.ExtRepo != "" {
		return g.ExtRepo
	}
	return defaultExtRepo
}

func (g *GitHub) apiBaseURL() string {
	if g != nil && g.apiBase != "" {
		return strings.TrimRight(g.apiBase, "/")
	}
	return githubAPIBase
}

func (g *GitHub) downloadBaseURL() string {
	if g != nil && g.downloadBase != "" {
		return strings.TrimRight(g.downloadBase, "/")
	}
	return githubDLBase
}
