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

const maxLogWindow = 7 * 24 * time.Hour

type logs struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	env           string
	since         string
	start         string
	end           string
	limit         int
	search        string
	order         string
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
	f.StringVar(&l.since, "since", "24h", "Trailing window from now (Go duration). e.g. 24h, 2h30m. Mutually exclusive with --start/--end.")
	f.StringVar(&l.start, "start", "", "Absolute window start (RFC3339, YYYY-MM-DDTHH:MM:SS, or YYYY-MM-DD). Mutually exclusive with --since.")
	f.StringVar(&l.end, "end", "", "Absolute window end (same formats as --start; defaults to 7 days after --start when omitted).")
	f.IntVar(&l.limit, "limit", 0, "How many logs to retrieve; 0 means no explicit limit")
	f.StringVar(&l.search, "search", "", "Search query to filter logs")
	f.StringVar(&l.order, "order", "asc", "Sort order for returned logs (asc or desc). Defaults to asc; the platform API defaults to desc.")

	_ = cmd.MarkFlagRequired("environment")

	return cmd
}

func (l *logs) Run(cmd *cobra.Command, args []string) {
	currentDir, err := os.Getwd()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get current directory").WithReason(err).Done()
	}

	if err := validateOrder(l.order); err != nil {
		utils.NewExitError().WithMessage(err.Error()).Done()
	}

	projectId := l.projectId.GetFlagOrDie(currentDir)
	start, end := l.resolveTimeRange(cmd)

	resp := LogsAPIResponse{}
	req := l.newLogsRequest(projectId, &resp).
		WithQueryParam("start", start).
		WithQueryParam("end", end)
	if err := req.Do(); err != nil {
		utils.NewExitError().WithMessage("failed to retrieve logs").WithReason(err).Done()
	}

	printLogs(resp.Logs)
}

func (l *logs) resolveTimeRange(cmd *cobra.Command) (string, string) {
	format := func(t time.Time) string { return t.Truncate(time.Millisecond).Format(time.RFC3339Nano) }

	hasStart := l.start != ""
	hasEnd := l.end != ""
	sinceChanged := cmd.Flags().Changed("since")

	if (hasStart || hasEnd) && sinceChanged {
		utils.NewExitError().WithMessage("--start/--end and --since are mutually exclusive").Done()
	}

	if hasStart || hasEnd {
		if !hasStart {
			utils.NewExitError().WithMessage("--start is required when --end is provided").Done()
		}
		start := parseLogTimestamp("--start", l.start)
		end := start.Add(maxLogWindow)
		if hasEnd {
			end = parseLogTimestamp("--end", l.end)
		}
		if !end.After(start) {
			utils.NewExitError().WithMessage("--end must be after --start").Done()
		}
		return format(start), format(end)
	}

	// --since mode (default 24h)
	durStr := l.since
	if durStr == "" {
		durStr = "24h"
	}
	dur, err := time.ParseDuration(durStr)
	if err != nil {
		utils.NewExitError().WithMessage("invalid --since duration, must be a valid Go duration").WithReason(err).Done()
	}
	if dur < 0 {
		dur = 0
	}

	now := time.Now().UTC()
	return format(now.Add(-dur)), format(now)
}

// parseLogTimestamp parses a user-supplied timestamp flag value. Accepts RFC3339,
// YYYY-MM-DDTHH:MM:SS, or YYYY-MM-DD. Exits with a clear error on invalid input.
func parseLogTimestamp(flag, value string) time.Time {
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC()
		}
	}
	utils.NewExitError().WithMessage(
		fmt.Sprintf("invalid %s %q: use RFC3339 (e.g. 2026-06-24T00:00:00Z) or YYYY-MM-DD", flag, value),
	).Done()
	return time.Time{}
}

// validateOrder accepts only the platform API's order values.
func validateOrder(order string) error {
	switch order {
	case "asc", "desc":
		return nil
	default:
		return fmt.Errorf("invalid --order %q (must be one of [asc desc])", order)
	}
}

// newLogsRequest builds the base HTTP request with common query params
func (l *logs) newLogsRequest(projectId string, into any) *utils.HTTPRequest {
	requestPath := "o/:organisation/projects/" + projectId + "/logs"
	req := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(into).
		WithPath(requestPath)
	return l.withLogQueryParams(req)
}

// withLogQueryParams attaches the shared logs query parameters.
func (l *logs) withLogQueryParams(req *utils.HTTPRequest) *utils.HTTPRequest {
	req = req.
		WithQueryParam("project_env", l.env).
		WithQueryParam("order", l.order).
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
