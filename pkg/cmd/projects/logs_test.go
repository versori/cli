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
	"strconv"
	"testing"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

func TestLogsOrderFlagDefault(t *testing.T) {
	cmd := NewLogs(config.NewConfigFactory())
	flag := cmd.Flags().Lookup("order")
	if flag == nil {
		t.Fatal("expected --order flag to be registered")
	}
	if got := flag.DefValue; got != "asc" {
		t.Fatalf("--order default = %q, want %q", got, "asc")
	}
}

func TestValidateOrder(t *testing.T) {
	tests := []struct {
		name    string
		order   string
		wantErr bool
	}{
		{name: "asc", order: "asc"},
		{name: "desc", order: "desc"},
		{name: "empty", order: "", wantErr: true},
		{name: "invalid", order: "newest", wantErr: true},
		{name: "wrong case", order: "ASC", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOrder(tt.order)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCollectLogPagesFollowsNextToken(t *testing.T) {
	calls := 0
	logs, warning, err := collectLogPages(func(start, end, token string, first int) (LogsAPIResponse, error) {
		calls++
		if calls == 1 {
			if token != "" || first != maxAPILogPage {
				t.Fatalf("first page query = token %q first %d", token, first)
			}
			return LogsAPIResponse{
				Logs:      []RawLog{{Timestamp: "2026-11-01T00:00:00Z", Message: "a"}},
				NextToken: "page-2",
			}, nil
		}
		if token != "page-2" || start != "2026-11-01T00:00:00Z" {
			t.Fatalf("second page query = start %q token %q", start, token)
		}
		return LogsAPIResponse{
			Logs: []RawLog{{Timestamp: "2026-11-01T00:00:01Z", Message: "b"}},
		}, nil
	}, "2026-11-01T00:00:00Z", "2026-11-02T00:00:00Z", "asc", 2000)
	if err != nil {
		t.Fatal(err)
	}
	if warning != "" {
		t.Fatalf("warning = %q", warning)
	}
	if calls != 2 || len(logs) != 2 || logs[1].Message != "b" {
		t.Fatalf("calls=%d logs=%+v", calls, logs)
	}
}

func TestCollectLogPagesSlidesTimestampWithoutToken(t *testing.T) {
	calls := 0
	logs, warning, err := collectLogPages(func(start, end, token string, first int) (LogsAPIResponse, error) {
		calls++
		if token != "" {
			t.Fatalf("token = %q", token)
		}
		if calls == 1 {
			page := make([]RawLog, maxAPILogPage)
			for i := range page {
				page[i] = RawLog{Timestamp: "2026-11-01T00:00:00Z", Message: "old"}
			}
			page[len(page)-1] = RawLog{Timestamp: "2026-11-01T01:00:00Z", Message: "edge"}
			return LogsAPIResponse{Logs: page}, nil
		}
		if start != "2026-11-01T01:00:00Z" {
			t.Fatalf("slid start = %q", start)
		}
		return LogsAPIResponse{Logs: []RawLog{
			{Timestamp: "2026-11-01T01:00:00Z", Message: "edge"},
			{Timestamp: "2026-11-01T02:00:00Z", Message: "new"},
		}}, nil
	}, "2026-10-26T00:00:00Z", "2026-11-02T00:00:00Z", "asc", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if warning != "" || calls != 2 {
		t.Fatalf("calls=%d warning=%q", calls, warning)
	}
	if len(logs) != maxAPILogPage+1 || logs[len(logs)-1].Message != "new" {
		t.Fatalf("len=%d last=%+v", len(logs), logs[len(logs)-1])
	}
}

func TestCollectLogPagesStopsWhenTimestampCannotSplit(t *testing.T) {
	calls := 0
	page := make([]RawLog, maxAPILogPage)
	for i := range page {
		page[i] = RawLog{Timestamp: "2026-11-01T00:00:00Z", Message: "same"}
	}
	logs, warning, err := collectLogPages(func(start, end, token string, first int) (LogsAPIResponse, error) {
		calls++
		return LogsAPIResponse{Logs: page}, nil
	}, "2026-11-01T00:00:00Z", "2026-11-02T00:00:00Z", "asc", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(logs) != maxAPILogPage {
		t.Fatalf("calls=%d len=%d", calls, len(logs))
	}
	if warning == "" {
		t.Fatal("expected a warning when the page cannot be split")
	}
}

func TestCollectLogPagesDescSlidesEnd(t *testing.T) {
	calls := 0
	_, _, err := collectLogPages(func(start, end, token string, first int) (LogsAPIResponse, error) {
		calls++
		if calls == 1 {
			page := make([]RawLog, maxAPILogPage)
			for i := range page {
				page[i] = RawLog{Timestamp: "2026-11-02T00:00:00Z", Message: "new"}
			}
			page[len(page)-1] = RawLog{Timestamp: "2026-11-01T00:00:00Z", Message: "edge"}
			return LogsAPIResponse{Logs: page}, nil
		}
		if end != "2026-11-01T00:00:00Z" {
			t.Fatalf("slid end = %q", end)
		}
		return LogsAPIResponse{Logs: []RawLog{{Timestamp: "2026-10-26T00:00:00Z", Message: "older"}}}, nil
	}, "2026-10-20T00:00:00Z", "2026-11-02T00:00:00Z", "desc", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestCollectLogPagesHonoursLimitAcrossPages(t *testing.T) {
	calls := 0
	logs, warning, err := collectLogPages(func(start, end, token string, first int) (LogsAPIResponse, error) {
		calls++
		if calls == 1 && first != maxAPILogPage {
			t.Fatalf("first page size = %d", first)
		}
		if calls == 2 && first != 500 {
			t.Fatalf("second page size = %d", first)
		}
		page := make([]RawLog, first)
		for i := range page {
			page[i] = RawLog{Timestamp: "2026-11-01T00:00:0" + strconv.Itoa(calls) + "Z", Message: "m"}
		}
		page[len(page)-1].Timestamp = "2026-11-01T01:00:0" + strconv.Itoa(calls) + "Z"
		return LogsAPIResponse{Logs: page}, nil
	}, "2026-11-01T00:00:00Z", "2026-11-08T00:00:00Z", "asc", 1500)
	if err != nil {
		t.Fatal(err)
	}
	if warning != "" || calls != 2 || len(logs) != 1500 {
		t.Fatalf("calls=%d len=%d warning=%q", calls, len(logs), warning)
	}
}

func TestCollectLogPagesSinglePageDoesNotContinue(t *testing.T) {
	calls := 0
	logs, warning, err := collectLogPages(func(start, end, token string, first int) (LogsAPIResponse, error) {
		calls++
		page := make([]RawLog, maxAPILogPage)
		for i := range page {
			page[i] = RawLog{Timestamp: "2026-11-01T00:00:00Z", Message: "x"}
		}
		return LogsAPIResponse{Logs: page}, nil
	}, "2026-11-01T00:00:00Z", "2026-11-02T00:00:00Z", "asc", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(logs) != 1000 || warning == "" {
		t.Fatalf("calls=%d len=%d warning=%q", calls, len(logs), warning)
	}
}

func TestWithLogQueryParamsOrder(t *testing.T) {
	for _, order := range []string{"asc", "desc"} {
		t.Run(order, func(t *testing.T) {
			l := &logs{order: order, env: "production"}
			req := l.withLogQueryParams(utils.NewHTTPBuilder("http://example.test").New())
			if got := req.Query["order"]; got != order {
				t.Fatalf("Query[order] = %q, want %q", got, order)
			}
		})
	}
}
