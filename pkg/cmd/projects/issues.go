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
	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/issues"
)

// NewIssuesCommand creates the `versori projects issues` parent command — the project-scoped view of
// issues. Unlike the org-level `versori issues` (which spans every project), these require a project
// (from --project or .versori) and resolve --environment by name against that project.
func NewIssuesCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "issues",
		Aliases: []string{"issue"},
		Short:   "List, inspect, and update issues for a project",
	}

	cmd.AddCommand(
		issues.NewProjectList(c),
		issues.NewProjectGet(c),
		issues.NewProjectUpdate(c),
	)

	return cmd
}
