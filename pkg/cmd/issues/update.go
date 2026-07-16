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
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type update struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	projectScoped bool
	status        string
	severity      string
}

// NewUpdate builds the org-level `issues update <issue-id>` — change an issue's status (ack/resolve)
// or severity by its (globally unique) ID. This is a mutation: run it only when explicitly asked.
func NewUpdate(c *config.ConfigFactory) *cobra.Command { return newUpdate(c, false) }

// NewProjectUpdate builds the project-scoped `projects issues update <issue-id>`. Same mutation,
// but --project is required (defaults from .versori) so the correct org context is selected before
// the update is applied.
func NewProjectUpdate(c *config.ConfigFactory) *cobra.Command { return newUpdate(c, true) }

func newUpdate(c *config.ConfigFactory, projectScoped bool) *cobra.Command {
	u := &update{configFactory: c, projectScoped: projectScoped}

	cmd := &cobra.Command{
		Use:   "update <issue-id>",
		Short: "Update an issue's status or severity",
		Long: `Update the editable fields of an issue. Common uses:

  update <id> --status acked      # acknowledge
  update <id> --status resolved   # mark resolved (server records resolved_at)

Mirrors the platform: only the fields you pass are sent. This is a mutation — run it only when
explicitly asked.`,
		Args: cobra.ExactArgs(1),
		Run:  u.Run,
	}

	f := cmd.Flags()
	if projectScoped {
		u.projectId.SetFlag(f)
	}
	f.StringVar(&u.status, "status", "", "New status (open, closed, acked, resolved)")
	f.StringVar(&u.severity, "severity", "", "New severity (critical, high, low, medium)")

	return cmd
}

func (u *update) Run(_ *cobra.Command, args []string) {
	issueID := args[0]

	if u.projectScoped {
		// Resolves --project (or .versori) and switches to the owning org context before the PATCH.
		_ = u.projectId.GetFlagOrDie(".")
	}

	if u.status == "" && u.severity == "" {
		utils.NewExitError().WithMessage("nothing to update: pass at least one of --status, --severity").Done()
	}
	if err := validateStatus(u.status); err != nil {
		utils.NewExitError().WithReason(err).Done()
	}
	if err := validateSeverity(u.severity); err != nil {
		utils.NewExitError().WithReason(err).Done()
	}

	body := v1.UpdateIssue{}
	if u.status != "" {
		s := v1.IssueStatusEnum(u.status)
		body.Status = &s
	}
	if u.severity != "" {
		sev := v1.IssueSeverityEnum(u.severity)
		body.Severity = &sev
	}

	err := u.configFactory.
		NewRequest().
		WithMethod(http.MethodPatch).
		WithPath("o/:organisation/issues/" + issueID).
		JSONBody(body).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to update issue").WithReason(err).Done()
	}

	fmt.Println("issue " + issueID + " updated successfully")
}
