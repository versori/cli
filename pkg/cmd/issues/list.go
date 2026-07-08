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

package issues

import (
	"net/http"
	"strconv"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type list struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId

	// filters
	status        string
	environmentId string
	severity      string

	// pagination
	first  int
	after  string
	before string
}

// NewList builds `issues list` — enumerate issues for the current organisation. Read-only and safe
// to run for diagnosis. When run inside a synced project directory the project filter defaults from
// .versori; pass --project to override or list a different project, and omit both to list every
// project in the org.
func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues for the current organisation",
		Long: `List issues raised in the organisation (via ctx.createIssue(), unhandled workflow errors, or
platform events such as an out-of-memory kill).

Filters (all optional) narrow server-side by status, project, environment, and severity. Inside a
synced project directory the project filter defaults from .versori. An OOM-killed project surfaces
here as an issue titled "OOM Killed" — inspect it with 'issues get <id>'.`,
		Run: l.Run,
	}

	f := cmd.Flags()
	l.projectId.SetFlag(f)
	f.StringVar(&l.status, "status", "", "Filter by status (open, closed, acked, resolved)")
	f.StringVar(&l.environmentId, "environment", "", "Filter by environment ID")
	f.StringVar(&l.severity, "severity", "", "Filter by severity (critical, high, low, medium)")
	f.IntVar(&l.first, "first", 0, "Max issues to return (0 lets the server default apply)")
	f.StringVar(&l.after, "after", "", "Pagination cursor: pass a prior response's last issue ID to fetch the next page")
	f.StringVar(&l.before, "before", "", "Pagination cursor for the previous page")

	return cmd
}

func (l *list) Run(_ *cobra.Command, _ []string) {
	projectId := l.projectId.GetProjectIDFromDir(".")
	if projectId != "" {
		config.MaybeApplyVersoriContextForProject(".", projectId)
	}

	if err := validateStatus(l.status); err != nil {
		utils.NewExitError().WithReason(err).Done()
	}
	if err := validateSeverity(l.severity); err != nil {
		utils.NewExitError().WithReason(err).Done()
	}

	req := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		WithPath("o/:organisation/issues")

	if l.status != "" {
		req = req.WithQueryParam("status", l.status)
	}
	if projectId != "" {
		req = req.WithQueryParam("project_id", projectId)
	}
	if l.environmentId != "" {
		req = req.WithQueryParam("environment_id", l.environmentId)
	}
	if l.severity != "" {
		req = req.WithQueryParam("severity", l.severity)
	}
	if l.first > 0 {
		req = req.WithQueryParam("first", strconv.Itoa(l.first))
	}
	if l.after != "" {
		req = req.WithQueryParam("after", l.after)
	}
	if l.before != "" {
		req = req.WithQueryParam("before", l.before)
	}

	resp := []v1.Issue{}
	if err := req.Into(&resp).Do(); err != nil {
		utils.NewExitError().WithMessage("failed to list issues").WithReason(err).Done()
	}

	items := make([]printableIssue, 0, len(resp))
	for _, i := range resp {
		items = append(items, toPrintableIssue(i))
	}

	l.configFactory.Print(items)
}
