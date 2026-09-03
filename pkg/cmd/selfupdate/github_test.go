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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLITarballName(t *testing.T) {
	got := CLITarballName("v0.0.9", "darwin", "arm64")
	want := "cli_0.0.9_darwin_arm64.tar.gz"
	if got != want {
		t.Fatalf("CLITarballName = %q; want %q", got, want)
	}

	got = CLITarballName("0.0.9", "linux", "amd64")
	want = "cli_0.0.9_linux_amd64.tar.gz"
	if got != want {
		t.Fatalf("CLITarballName = %q; want %q", got, want)
	}

	got = CLITarballName("v0.0.9", "Darwin", "aarch64")
	want = "cli_0.0.9_darwin_arm64.tar.gz"
	if got != want {
		t.Fatalf("CLITarballName = %q; want %q", got, want)
	}
}

func TestPickVSIXAsset(t *testing.T) {
	got, err := PickVSIXAsset([]string{"notes.txt", "other.vsix", "versori-vscode-0.1.0.vsix"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "versori-vscode-0.1.0.vsix" {
		t.Fatalf("PickVSIXAsset preferred = %q", got)
	}

	got, err = PickVSIXAsset([]string{"checksums.txt", "extension.vsix"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "extension.vsix" {
		t.Fatalf("PickVSIXAsset unique = %q", got)
	}

	if _, err := PickVSIXAsset([]string{"a.vsix", "b.vsix"}); err == nil {
		t.Fatal("PickVSIXAsset multiple .vsix: want error")
	}
	if _, err := PickVSIXAsset([]string{"checksums.txt"}); err == nil {
		t.Fatal("PickVSIXAsset none: want error")
	}
}

func TestGitHubLatestTag(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/repos/versori/cli/releases/latest" {
				t.Errorf("path=%q", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(`{"tag_name":"v0.0.9"}`))
		}))
		t.Cleanup(srv.Close)

		g := &GitHub{Client: srv.Client(), apiBase: srv.URL}
		tag, err := g.LatestTag("versori/cli")
		if err != nil {
			t.Fatal(err)
		}
		if tag != "v0.0.9" {
			t.Fatalf("tag=%q", tag)
		}
	})

	t.Run("404", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)

		g := &GitHub{Client: srv.Client(), apiBase: srv.URL}
		if _, err := g.LatestTag("versori/cli"); err == nil {
			t.Fatal("LatestTag 404: want error")
		}
	})
}

func TestGitHubDownloadReleaseFile(t *testing.T) {
	const body = "tarball-bytes"
	const asset = "/versori/cli/releases/download/v0.0.9/cli_0.0.9_darwin_arm64.tar.gz"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != asset {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "cli_0.0.9_darwin_arm64.tar.gz")
	g := &GitHub{Client: srv.Client(), downloadBase: srv.URL}
	if err := g.DownloadReleaseFile("versori/cli", "v0.0.9", "cli_0.0.9_darwin_arm64.tar.gz", dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("dest=%q", got)
	}

	if err := g.DownloadReleaseFile("versori/cli", "v0.0.9", "missing.tar.gz", filepath.Join(t.TempDir(), "missing.tar.gz")); err == nil {
		t.Fatal("DownloadReleaseFile 404: want error")
	}
}

func TestVerifySHA256(t *testing.T) {
	dir := t.TempDir()
	filename := "cli_0.0.9_darwin_arm64.tar.gz"
	path := filepath.Join(dir, filename)
	payload := []byte("payload")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])

	okSums := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(okSums, []byte(hash+"  "+filename+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(path, okSums, filename); err != nil {
		t.Fatalf("match: %v", err)
	}

	badSums := filepath.Join(dir, "bad-checksums.txt")
	if err := os.WriteFile(badSums, []byte("deadbeef  "+filename+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(path, badSums, filename); err == nil {
		t.Fatal("mismatch: want error")
	}

	emptySums := filepath.Join(dir, "empty-checksums.txt")
	if err := os.WriteFile(emptySums, []byte("deadbeef  other.tar.gz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(path, emptySums, filename); err == nil {
		t.Fatal("missing filename: want error")
	}
}

func TestExtractVersoriBinary(t *testing.T) {
	dir := t.TempDir()
	tarball := filepath.Join(dir, "cli.tar.gz")
	writeTarGz(t, tarball, map[string][]byte{
		"versori": []byte("#!/bin/sh\necho versori\n"),
		"README":  []byte("hi"),
	})

	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ExtractVersoriBinary(tarball, dest)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dest, "versori")
	if got != want {
		t.Fatalf("binaryPath=%q; want %q", got, want)
	}
	body, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "versori") {
		t.Fatalf("extracted body=%q", body)
	}

	emptyTar := filepath.Join(dir, "empty.tar.gz")
	writeTarGz(t, emptyTar, map[string][]byte{"README": []byte("no binary")})
	if _, err := ExtractVersoriBinary(emptyTar, filepath.Join(dir, "empty-out")); err == nil {
		t.Fatal("ExtractVersoriBinary missing versori: want error")
	}
}

func TestGitHubDefaults(t *testing.T) {
	g := &GitHub{}
	if g.cliRepo() != "versori/cli" {
		t.Fatalf("CLIRepo default=%q", g.cliRepo())
	}
	if g.extRepo() != "versori/versori-vscode-extension" {
		t.Fatalf("ExtRepo default=%q", g.extRepo())
	}
	if g.apiBaseURL() != "https://api.github.com" {
		t.Fatalf("apiBase=%q", g.apiBaseURL())
	}
	if g.downloadBaseURL() != "https://github.com" {
		t.Fatalf("downloadBase=%q", g.downloadBaseURL())
	}
}

func writeTarGz(t *testing.T, dest string, files map[string][]byte) {
	t.Helper()
	f, err := os.Create(dest)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
