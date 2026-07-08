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
	"github.com/versori/cli/pkg/utils"
)

type update struct {
	configFactory    *config.ConfigFactory
	status           string
	resolutionStatus string
	severity         string
}

// NewUpdate builds `issues update <issue-id>` — change an issue's status (ack/resolve), resolution
// status, or severity. This is a mutation: it changes live issue state and should only be run when
// explicitly requested.
func NewUpdate(c *config.ConfigFactory) *cobra.Command {
	u := &update{configFactory: c}

	cmd := &cobra.Command{
		Use:   "update <issue-id>",
		Short: "Update an issue's status, resolution, or severity",
		Long: `Update the editable fields of an issue. Common uses:

  issues update <id> --status acked                          # acknowledge
  issues update <id> --status resolved --resolution-status resolved

Only the flags you pass are changed; everything else is left as-is. This is a mutation — run it only
when explicitly asked.`,
		Args: cobra.ExactArgs(1),
		Run:  u.Run,
	}

	f := cmd.Flags()
	f.StringVar(&u.status, "status", "", "New status (open, closed, acked, resolved)")
	f.StringVar(&u.resolutionStatus, "resolution-status", "", "Resolution status (resolved, negated, ignored)")
	f.StringVar(&u.severity, "severity", "", "New severity (critical, high, low, medium)")

	return cmd
}

func (u *update) Run(_ *cobra.Command, args []string) {
	issueID := args[0]

	if u.status == "" && u.resolutionStatus == "" && u.severity == "" {
		utils.NewExitError().WithMessage("nothing to update: pass at least one of --status, --resolution-status, --severity").Done()
	}
	if err := validateStatus(u.status); err != nil {
		utils.NewExitError().WithReason(err).Done()
	}
	if err := validateResolutionStatus(u.resolutionStatus); err != nil {
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
	if u.resolutionStatus != "" {
		rs := v1.IssueResolutionStatusEnum(u.resolutionStatus)
		body.ResolutionStatus = &rs
	}
	if u.severity != "" {
		sev := v1.IssueSeverityEnum(u.severity)
		body.Severity = &sev
	}

	err := u.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		WithPath("o/:organisation/issues/" + issueID).
		JSONBody(body).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to update issue").WithReason(err).Done()
	}

	fmt.Println("issue " + issueID + " updated successfully")
}
