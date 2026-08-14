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
