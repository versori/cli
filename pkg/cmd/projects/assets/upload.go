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

	"github.com/spf13/cobra"
	"github.com/versori/cli/pkg/cmd/flags"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type signedURLRequest struct {
	ContentType   string `json:"contentType"`
	ContentLength int64  `json:"contentLength"`
	Filename      string `json:"filename"`
	Folder        string `json:"folder,omitempty"`
}

type signedURLResponse struct {
	UploadURL string `json:"uploadURL"`
	URL       string `json:"url"`
}

type upload struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	file          string
}

func NewUpload(c *config.ConfigFactory) *cobra.Command {
	u := &upload{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "upload --file <path>",
		Short: "Upload an asset file to the Versori platform",
		Long:  "Upload uploads a file as an asset to the Versori platform. Assets can be used as context by Versori AI agents.",
		Run:   u.Run,
	}

	flags := cmd.Flags()
	u.projectId.SetFlag(flags)
	flags.StringVarP(&u.file, "file", "f", "", "Path to the file to upload (required)")

	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func (u *upload) Run(cmd *cobra.Command, args []string) {
	projectId := u.projectId.GetFlagOrDie(".")
	orgId := u.configFactory.Context.OrganisationId

	// Stat the file to get content length and derive metadata.
	info, err := os.Stat(u.file)
	if err != nil {
		utils.NewExitError().WithMessage("failed to stat file").WithReason(err).Done()
	}

	filename := filepath.Base(u.file)
	contentType := ContentTypeFromFilename(filename)
	if contentType == "" {
		utils.NewExitError().
			WithMessage(fmt.Sprintf("unsupported file type %q — allowed types: PDF, TXT, Markdown, JSON, YAML, JPEG, PNG, GIF, WebP", filepath.Ext(filename))).
			Done()
	}

	body := signedURLRequest{
		ContentType:   contentType,
		ContentLength: info.Size(),
		Filename:      filename,
		Folder:        "research/documents",
	}

	requestPath := "assets/organisations/" + orgId + "/" + projectId + "/signed-url"

	var signedURL signedURLResponse

	err = u.configFactory.
		NewAIRequest().
		WithMethod(http.MethodPost).
		JSONBody(body).
		Into(&signedURL).
		WithPath(requestPath).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get signed URL").WithReason(err).Done()
	}

	// Open the file for uploading.
	f, err := os.Open(u.file)
	if err != nil {
		utils.NewExitError().WithMessage("failed to open file for upload").WithReason(err).Done()
	}
	defer f.Close() //nolint:errcheck

	req, err := http.NewRequest(http.MethodPut, signedURL.UploadURL, f)
	if err != nil {
		utils.NewExitError().WithMessage("failed to create upload request").WithReason(err).Done()
	}

	req.ContentLength = info.Size()
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		utils.NewExitError().WithMessage("failed to upload file").WithReason(err).Done()
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		utils.NewExitError().
			WithMessage(fmt.Sprintf("upload failed with status %d: %s", resp.StatusCode, string(b))).
			Done()
	}

	fmt.Printf("Successfully uploaded %q\n", filename)
	fmt.Printf("Asset URL: %s\n", signedURL.URL)
}

// allowedContentTypes maps file extensions to the MIME types accepted by the
// Versori asset API: PDF, TXT, Markdown, JSON, YAML, JPEG, PNG, GIF, WebP.
var allowedContentTypes = map[string]string{
	".pdf":      "application/pdf",
	".txt":      "text/plain",
	".md":       "text/markdown",
	".markdown": "text/markdown",
	".json":     "application/json",
	".yaml":     "text/yaml",
	".yml":      "text/yaml",
	".jpg":      "image/jpeg",
	".jpeg":     "image/jpeg",
	".png":      "image/png",
	".gif":      "image/gif",
	".webp":     "image/webp",
}

// ContentTypeFromFilename returns the MIME type for the file extension.
// Returns an empty string when the extension is not in the allowed set.
func ContentTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	return allowedContentTypes[ext]
}
