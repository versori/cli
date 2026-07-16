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

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

// maxGetPages caps how many list pages `issues get` walks looking for the target ID. The platform
// exposes no GET-by-id endpoint, so we page the issue list and match client-side; the cap bounds the
// work for a very large backlog. Narrow with --project (or use the project-scoped variant) to reach
// older issues faster.
const maxGetPages = 100

type get struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	projectScoped bool
	env           string
}

// NewGet builds the org-level `issues get <issue-id>` — fetch a single issue's full detail by
// walking the org's issue list (newest first). --project is an optional filter that scopes the walk.
func NewGet(c *config.ConfigFactory) *cobra.Command { return newGet(c, false) }

// NewProjectGet builds the project-scoped `projects issues get <issue-id>`. --project is required
// (defaults from .versori) and --environment is an environment name resolved against the project.
func NewProjectGet(c *config.ConfigFactory) *cobra.Command { return newGet(c, true) }

func newGet(c *config.ConfigFactory, projectScoped bool) *cobra.Command {
	g := &get{configFactory: c, projectScoped: projectScoped}

	cmd := &cobra.Command{
		Use:   "get <issue-id>",
		Short: "Get full detail for a single issue",
		Args:  cobra.ExactArgs(1),
		Run:   g.Run,
	}

	g.projectId.SetFlag(cmd.Flags())
	if projectScoped {
		cmd.Flags().StringVar(&g.env, "environment", "", "Environment name to filter by (auto-selected when the project has exactly one environment)")
	}

	return cmd
}

func (g *get) Run(_ *cobra.Command, args []string) {
	target := args[0]

	projectId, envId := g.resolveScope()

	after := ""
	for page := 0; page < maxGetPages; page++ {
		req := g.configFactory.
			NewRequest().
			WithMethod(http.MethodGet).
			WithPath("o/:organisation/issues")

		if projectId != "" {
			req = req.WithQueryParam("project_id", projectId)
		}
		if envId != "" {
			req = req.WithQueryParam("environment_id", envId)
		}
		if after != "" {
			req = req.WithQueryParam("after", after)
		}

		resp := []v1.Issue{}
		if err := req.Into(&resp).Do(); err != nil {
			utils.NewExitError().WithMessage("failed to list issues").WithReason(err).Done()
		}

		if len(resp) == 0 {
			break
		}

		for _, i := range resp {
			if i.Id.String() == target {
				g.configFactory.Print(toPrintableIssueDetail(i))

				return
			}
		}

		// Advance the cursor to the last item on this page. If it hasn't moved the server isn't
		// paginating on this cursor, so stop rather than loop forever.
		next := resp[len(resp)-1].Id.String()
		if next == after {
			break
		}
		after = next
	}

	utils.NewExitError().WithMessage("issue " + target + " not found (try --project to narrow the search, or the issue may be older than the pages scanned)").Done()
}

// resolveScope mirrors list.resolveScope: org-level treats --project as an optional filter and
// --environment as a raw ID; project-scoped requires --project and resolves --environment by name.
func (g *get) resolveScope() (projectId, envId string) {
	if g.projectScoped {
		projectId = g.projectId.GetFlagOrDie(".")
		return projectId, resolveEnvironmentId(g.configFactory, projectId, g.env)
	}

	projectId = g.projectId.GetProjectIDFromDir(".")
	if projectId != "" {
		config.MaybeApplyVersoriContextForProject(".", projectId)
	}

	return projectId, g.env
}
