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
	projectScoped bool

	// filters
	status   string
	env      string
	severity string

	// pagination
	first  int
	after  string
	before string
}

// NewList builds the org-level `issues list` — enumerate issues across the whole organisation.
// This is the platform view: `--project` is an optional filter and `--environment` filters by
// environment ID. Read-only and safe to run for diagnosis.
func NewList(c *config.ConfigFactory) *cobra.Command { return newList(c, false) }

// NewProjectList builds the project-scoped `projects issues list`. `--project` is required (defaults
// from .versori inside a synced directory) and `--environment` is an environment *name* resolved
// against that project — auto-selected when the project has exactly one environment.
func NewProjectList(c *config.ConfigFactory) *cobra.Command { return newList(c, true) }

func newList(c *config.ConfigFactory, projectScoped bool) *cobra.Command {
	l := &list{configFactory: c, projectScoped: projectScoped}

	cmd := &cobra.Command{
		Use: "list",
		Run: l.Run,
	}

	f := cmd.Flags()
	l.projectId.SetFlag(f)
	f.StringVar(&l.status, "status", "", "Filter by status (open, closed, acked, resolved)")
	f.StringVar(&l.severity, "severity", "", "Filter by severity (critical, high, low, medium)")
	f.IntVar(&l.first, "first", 0, "Max issues to return (0 lets the server default apply)")
	f.StringVar(&l.after, "after", "", "Pagination cursor: pass a prior response's last issue ID to fetch the next page")
	f.StringVar(&l.before, "before", "", "Pagination cursor for the previous page")

	if projectScoped {
		cmd.Short = "List issues for a project"
		cmd.Long = `List issues raised for a single project (via ctx.createIssue(), unhandled workflow errors, or
platform events such as an out-of-memory kill).

--project is required; it defaults from .versori inside a synced project directory. --environment is
an environment name resolved against the project, and is auto-selected when the project has exactly
one environment.`
		f.StringVar(&l.env, "environment", "", "Environment name to filter by (auto-selected when the project has exactly one environment)")
	} else {
		cmd.Short = "List issues for the current organisation"
		cmd.Long = `List issues raised across the organisation (via ctx.createIssue(), unhandled workflow errors, or
platform events such as an out-of-memory kill).

Filters (all optional) narrow server-side by status, project, environment, and severity. Inside a
synced project directory the project filter defaults from .versori. An OOM-killed project surfaces
here as an issue titled "OOM Killed" — inspect it with 'issues get <id>'.`
		f.StringVar(&l.env, "environment", "", "Filter by environment ID")
	}

	return cmd
}

func (l *list) Run(_ *cobra.Command, _ []string) {
	if err := validateStatus(l.status); err != nil {
		utils.NewExitError().WithReason(err).Done()
	}
	if err := validateSeverity(l.severity); err != nil {
		utils.NewExitError().WithReason(err).Done()
	}

	projectId, envId := l.resolveScope()

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
	if envId != "" {
		req = req.WithQueryParam("environment_id", envId)
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

// resolveScope returns the project ID and environment ID to filter by. Org-level (project optional,
// env is a raw ID) and project-scoped (project required, env is a name resolved against the project)
// differ only here, so the rest of Run stays shared.
func (l *list) resolveScope() (projectId, envId string) {
	if l.projectScoped {
		projectId = l.projectId.GetFlagOrDie(".")
		return projectId, resolveEnvironmentId(l.configFactory, projectId, l.env)
	}

	projectId = l.projectId.GetProjectIDFromDir(".")
	if projectId != "" {
		config.MaybeApplyVersoriContextForProject(".", projectId)
	}

	return projectId, l.env
}
