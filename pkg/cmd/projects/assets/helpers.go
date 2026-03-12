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

package assets

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/versori/cli/pkg/cmd/config"
)

const DefaultAssetsDir = "versori-research"

// AssetsResponse is the API response returned when listing project assets.
type AssetsResponse struct {
	Assets     []Asset `json:"assets"`
	TotalCount int     `json:"totalCount"`
	HasMore    bool    `json:"hasMore"`
}

// Asset represents a single asset returned by the Versori platform.
type Asset struct {
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	LastModified string `json:"lastModified"`
	ContentType  string `json:"contentType"`
	Path         string `json:"path"`
	DownloadURL  string `json:"downloadUrl"`
}

// ListAssets fetches all assets for the given organisation and project.
func ListAssets(cf *config.ConfigFactory, orgId, projectId string) (AssetsResponse, error) {
	requestPath := "assets/organisations/" + orgId + "/" + projectId

	var resp AssetsResponse

	err := cf.
		NewAIRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath(requestPath).
		Do()

	return resp, err
}

// DownloadAssetToFile downloads the asset at downloadURL and writes it to
// directory/name. The directory is created if it does not exist.
func DownloadAssetToFile(downloadURL, name, directory string) error {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	//nolint:noctx
	httpResp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}
	defer httpResp.Body.Close() //nolint:errcheck

	if httpResp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("download failed with status %d: %s", httpResp.StatusCode, string(b))
	}

	outPath := filepath.Join(directory, name)

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	if _, err := io.Copy(f, httpResp.Body); err != nil {
		return fmt.Errorf("failed to write asset to file: %w", err)
	}

	return nil
}

// UploadAssetFile uploads a single file as a project asset via the signed-URL
// flow. It mirrors the logic used by the `asset upload` subcommand.
func UploadAssetFile(cf *config.ConfigFactory, orgId, projectId, filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	filename := filepath.Base(filePath)
	contentType := ContentTypeFromFilename(filename)
	if contentType == "" {
		return fmt.Errorf("unsupported file type %q — allowed types: PDF, TXT, Markdown, JSON, YAML, JPEG, PNG, GIF, WebP", filepath.Ext(filename))
	}

	body := signedURLRequest{
		ContentType:   contentType,
		ContentLength: info.Size(),
		Filename:      filename,
		Folder:        "research/documents",
	}

	requestPath := "assets/organisations/" + orgId + "/" + projectId + "/signed-url"

	var signedURL signedURLResponse

	err = cf.
		NewAIRequest().
		WithMethod(http.MethodPost).
		JSONBody(body).
		Into(&signedURL).
		WithPath(requestPath).
		Do()
	if err != nil {
		return fmt.Errorf("failed to get signed URL for %s: %w", filename, err)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close() //nolint:errcheck

	req, err := http.NewRequest(http.MethodPut, signedURL.UploadURL, f)
	if err != nil {
		return fmt.Errorf("failed to create upload request for %s: %w", filename, err)
	}

	req.ContentLength = info.Size()
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file %s: %w", filename, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload of %s failed with status %d: %s", filename, resp.StatusCode, string(b))
	}

	return nil
}

// CollectAssetFiles walks the given directory and returns the paths of all
// files whose extension is in the allowed content-type set.
func CollectAssetFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read assets directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if _, ok := allowedContentTypes[ext]; !ok {
			continue
		}

		files = append(files, filepath.Join(dir, entry.Name()))
	}

	return files, nil
}
