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
	f.IntVar(&l.limit, "limit", 0, "How many logs to print. 0 returns 500. Any larger value is fetched in pages of at most 1000, with no maximum.")
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

	logs, warning, err := collectLogPages(func(start, end, token string, first int) (LogsAPIResponse, error) {
		return l.fetchPage(projectId, start, end, token, first)
	}, start, end, l.order, l.limit)
	if err != nil {
		utils.NewExitError().WithMessage("failed to retrieve logs").WithReason(err).Done()
	}

	printLogs(logs)
	if warning != "" {
		fmt.Fprintln(os.Stderr, warning)
	}
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

// fetchPage requests one page. first <= 0 omits the page size so the server uses its default.
func (l *logs) fetchPage(projectId, start, end, token string, first int) (LogsAPIResponse, error) {
	resp := LogsAPIResponse{}
	req := l.newLogsRequest(projectId, &resp).
		WithQueryParam("start", start).
		WithQueryParam("end", end)
	if token != "" {
		req = req.WithQueryParam("next_token", token)
	}
	if first > 0 {
		req = req.WithQueryParam("first", strconv.Itoa(first))
	}
	if err := req.Do(); err != nil {
		return LogsAPIResponse{}, err
	}

	return resp, nil
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

	return req
}

const maxAPILogPage = 1000

type logPageFetch func(start, end, token string, first int) (LogsAPIResponse, error)

// collectLogPages reads a window. --limit of 0 or 1–1000 is one request.
// A higher --limit keeps reading, in pages of at most 1000, until that many
// lines are printed or the window runs out. It follows nextToken when the API
// returns one, and when a full page has no token it continues from the last
// line's timestamp. That bound is inclusive, and a backend with no cursor can
// repeat the page, so lines already printed are dropped. A full page that
// makes no progress cannot be split; the warning says so.
func collectLogPages(fetch logPageFetch, start, end, order string, limit int) ([]RawLog, string, error) {
	windowStart, windowEnd := start, end
	token := ""
	slid := false
	stuckAt := ""
	emitted := map[string]struct{}{}
	var out []RawLog
	oneShot := limit <= maxAPILogPage

	for page := 0; page < 10000; page++ {
		first := maxAPILogPage
		if oneShot {
			first = limit
		} else if limit > 0 {
			remaining := limit - len(out)
			if remaining <= 0 {
				break
			}
			if remaining < first {
				first = remaining
			}
		}

		resp, err := fetch(windowStart, windowEnd, token, first)
		if err != nil {
			return nil, "", err
		}
		if len(resp.Logs) == 0 {
			return out, "", nil
		}

		logs := resp.Logs
		if slid {
			logs = dropEmitted(emitted, logs)
			if len(logs) == 0 {
				return out, fmt.Sprintf("stopped paging at %s: this timestamp filled a full page and the log backend returned no next-page token, so the rest of it cannot be fetched", stuckAt), nil
			}
		}

		if limit > 0 && len(out)+len(logs) > limit {
			logs = logs[:limit-len(out)]
		}
		rememberLogs(emitted, logs)
		out = append(out, logs...)

		if oneShot {
			if first == maxAPILogPage && len(resp.Logs) == maxAPILogPage {
				return out, "showing 1000 logs; pass a higher --limit to keep reading this window", nil
			}
			return out, "", nil
		}
		if limit > 0 && len(out) >= limit {
			return out, "", nil
		}

		if resp.NextToken != "" {
			token = resp.NextToken
			slid = false
			continue
		}

		if len(resp.Logs) < first {
			return out, "", nil
		}

		edge := resp.Logs[len(resp.Logs)-1].Timestamp
		token = ""
		slid = true
		stuckAt = edge
		if order == "desc" {
			windowEnd = edge
		} else {
			windowStart = edge
		}
	}

	return out, "stopped paging after 10000 pages", nil
}

func rememberLogs(emitted map[string]struct{}, logs []RawLog) {
	for _, l := range logs {
		emitted[rawLogKey(l)] = struct{}{}
	}
}

func dropEmitted(emitted map[string]struct{}, logs []RawLog) []RawLog {
	if len(emitted) == 0 {
		return logs
	}
	out := make([]RawLog, 0, len(logs))
	for _, l := range logs {
		if _, ok := emitted[rawLogKey(l)]; ok {
			continue
		}
		out = append(out, l)
	}
	return out
}

func rawLogKey(l RawLog) string {
	b, err := json.Marshal(l)
	if err != nil {
		return l.Timestamp + "\n" + l.Severity + "\n" + l.Message
	}
	return string(b)
}

func printLogs(logs []RawLog) {
	for _, l := range logs {
		b, _ := json.Marshal(l)
		_, _ = fmt.Fprintln(os.Stdout, string(b))
	}
}
