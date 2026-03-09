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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type logs struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	env           string
	since         string
	limit         int
	search        string
}

type LogsAPIResponse struct {
	Logs      []RawLog `json:"logs"`
	NextToken string   `json:"nextToken"`
}

type RawLog struct {
	Fields    map[string]any `json:"fields"`
	Message   string         `json:"message"`
	Severity  string         `json:"severity"`
	Timestamp string         `json:"timestamp"`
	Error     string         `json:"error"`
}

func NewLogs(c *config.ConfigFactory) *cobra.Command {
	l := &logs{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "logs ",
		Short: "Check the project logs",
		Run:   l.Run,
	}

	f := cmd.Flags()
	l.projectId.SetFlag(f)

	f.StringVar(&l.env, "environment", "", "The environment to retrieve logs for. e.g. (production, staging)")
	f.StringVar(&l.since, "since", "24h", "Go duration since now, e.g. 24h, 2h30m (default: 24h)")
	f.IntVar(&l.limit, "limit", 0, "How many logs to retrieve; 0 means no explicit limit")
	f.StringVar(&l.search, "search", "", "Search query to filter logs")

	_ = cmd.MarkFlagRequired("environment")

	return cmd
}

func (l *logs) Run(cmd *cobra.Command, args []string) {
	currentDir, err := os.Getwd()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get current directory").WithReason(err).Done()
	}

	projectId := l.projectId.GetFlagOrDie(currentDir)
	start, end := l.resolveTimeRange()

	resp := LogsAPIResponse{}
	req := l.newLogsRequest(projectId, &resp).
		WithQueryParam("start", start).
		WithQueryParam("end", end)
	if err := req.Do(); err != nil {
		utils.NewExitError().WithMessage("failed to retrieve logs").WithReason(err).Done()
	}

	printLogs(resp.Logs)
}

func (l *logs) resolveTimeRange() (string, string) {
	// Determine start and end based on --since duration and now
	durStr := l.since
	if durStr == "" {
		durStr = "24h"
	}
	dur, err := time.ParseDuration(durStr)
	if err != nil {
		utils.NewExitError().WithMessage("invalid --since duration, must be a valid Go duration").WithReason(err).Done()
	}
	if dur < 0 {
		// treat negative durations as zero to avoid future start times
		dur = 0
	}

	now := time.Now().UTC()
	start := now.Add(-dur)

	// round to ms to keep URLs tidy
	format := func(t time.Time) string { return t.Truncate(time.Millisecond).Format(time.RFC3339Nano) }

	return format(start), format(now)
}

// newLogsRequest builds the base HTTP request with common query params
func (l *logs) newLogsRequest(projectId string, into any) *utils.HTTPRequest {
	requestPath := "o/:organisation/projects/" + projectId + "/logs"
	req := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(into).
		WithPath(requestPath).
		WithQueryParam("project_env", l.env).
		WithQueryParam("order", "asc").
		WithQueryParam("latest", fmt.Sprintf("%t", false))
	if l.search != "" {
		req = req.WithQueryParam("search", l.search)
	}
	if l.limit > 0 {
		req = req.WithQueryParam("first", strconv.Itoa(l.limit))
	}

	return req
}

func printLogs(logs []RawLog) {
	for _, l := range logs {
		b, _ := json.Marshal(l)
		_, _ = fmt.Fprintln(os.Stdout, string(b))
	}
}
