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

package projects

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type proxy struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	method        string
	headers       []string
	queryParams   []string
	body          string
	urlPath       string
	env           string
}

type callIntegrationRequest struct {
	URLPath string              `json:"urlPath"`
	Method  string              `json:"method"`
	Headers map[string][]string `json:"headers,omitempty"`
	Query   map[string][]string `json:"query,omitempty"`
	Body    string              `json:"body,omitempty"`
}

type callIntegrationResponse struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

func NewProxy(c *config.ConfigFactory) *cobra.Command {
	p := &proxy{configFactory: c}

	cmd := &cobra.Command{
		Use:   "proxy --path <path> [flags]",
		Short: "Send an HTTP request to a project's deployed environment",
		Long: `Proxy sends an HTTP request to a project's deployed environment and returns
the response. This allows triggering a project remotely via an HTTP call.

The --body flag accepts a raw string or @filename to read the body from a file.
When using @filename, the Content-Type header is automatically inferred from
the file extension unless explicitly provided via --header.`,
		Run: p.Run,
	}

	f := cmd.Flags()
	p.projectId.SetFlag(f)

	f.StringVarP(&p.method, "method", "m", "POST", "HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)")
	f.StringArrayVarP(&p.headers, "header", "H", nil, "HTTP headers as key:value pairs (repeatable)")
	f.StringArrayVarP(&p.queryParams, "query", "q", nil, "Query parameters as key:value pairs (repeatable)")
	f.StringVarP(&p.body, "body", "b", "", "Request body (string or @filename to read from file)")
	f.StringVar(&p.urlPath, "path", "", "URL path to call on the integration (required)")
	f.StringVar(&p.env, "environment", "", "Project environment name (e.g. production, staging)")

	_ = cmd.MarkFlagRequired("path")
	_ = cmd.MarkFlagRequired("environment")

	return cmd
}

func (p *proxy) Run(cmd *cobra.Command, args []string) {
	currentDir, err := os.Getwd()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get current directory").WithReason(err).Done()
	}

	projectId := p.projectId.GetFlagOrDie(currentDir)
	reqHeaders := parseKeyValuePairs(p.headers, "header")
	reqQuery := parseKeyValuePairs(p.queryParams, "query")

	var bodyB64 string
	if p.body != "" {
		bodyBytes, contentType := resolveBody(p.body)

		bodyB64 = base64.StdEncoding.EncodeToString(bodyBytes)
		if contentType != "" {
			if _, ok := reqHeaders["Content-Type"]; !ok {
				reqHeaders["Content-Type"] = []string{contentType}
			}
		}
	}

	payload := callIntegrationRequest{
		URLPath: p.urlPath,
		Method:  strings.ToUpper(p.method),
		Headers: reqHeaders,
		Query:   reqQuery,
		Body:    bodyB64,
	}

	requestPath := "o/:organisation/projects/" + projectId + "/environments/call"
	resp := callIntegrationResponse{}

	req := p.configFactory.NewRequest().
		WithMethod(http.MethodPost).
		Into(&resp).
		WithPath(requestPath).
		JSONBody(payload).
		WithQueryParam("project_env", p.env)

	if err := req.Do(); err != nil {
		utils.NewExitError().WithMessage("failed to call integration").WithReason(err).Done()
	}

	printHTTPResponse(resp)
}

func parseKeyValuePairs(pairs []string, flagName string) map[string][]string {
	result := make(map[string][]string)
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			utils.NewExitError().WithMessage(fmt.Sprintf("invalid --%s format %q: expected key:value", flagName, pair)).Done()
		}
		result[strings.TrimSpace(parts[0])] = append(result[strings.TrimSpace(parts[0])], strings.TrimSpace(parts[1]))
	}

	return result
}

func resolveBody(body string) ([]byte, string) {
	if strings.HasPrefix(body, "@") {
		filePath := strings.TrimPrefix(body, "@")
		data, err := os.ReadFile(filePath)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read body file " + filePath).WithReason(err).Done()
		}

		return data, inferContentType(filePath)
	}

	return []byte(body), ""
}

func inferContentType(filePath string) string {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".txt":
		return "text/plain"
	case ".csv":
		return "text/csv"
	case ".yaml", ".yml":
		return "application/yaml"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

func printHTTPResponse(resp callIntegrationResponse) {
	statusText := http.StatusText(resp.Status)
	if statusText == "" {
		statusText = "Unknown"
	}

	fmt.Printf("HTTP %d %s\n", resp.Status, statusText)

	keys := make([]string, 0, len(resp.Headers))
	for k := range resp.Headers {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range resp.Headers[k] {
			fmt.Printf("%s: %s\n", k, v)
		}
	}

	fmt.Println() // blank line between headers and body

	if resp.Body == "" {
		return
	}

	bodyBytes, err := base64.StdEncoding.DecodeString(resp.Body)
	if err != nil {
		fmt.Println(resp.Body)

		return
	}

	contentType := ""
	for k, v := range resp.Headers {
		if strings.EqualFold(k, "content-type") && len(v) > 0 {
			contentType = strings.ToLower(v[0])

			break
		}
	}

	if contentType == "" {
		fmt.Println(resp.Body)

		return
	}

	if strings.Contains(contentType, "json") {
		var pretty json.RawMessage
		if json.Unmarshal(bodyBytes, &pretty) == nil {
			formatted, fmtErr := json.MarshalIndent(pretty, "", "  ")
			if fmtErr == nil {
				fmt.Println(string(formatted))

				return
			}
		}
	}

	if strings.Contains(contentType, "text") ||
		strings.Contains(contentType, "xml") ||
		strings.Contains(contentType, "html") ||
		strings.Contains(contentType, "yaml") {
		fmt.Println(string(bodyBytes))

		return
	}

	fmt.Println(resp.Body)
}
