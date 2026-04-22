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
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type star struct {
	configFactory *config.ConfigFactory
}

func NewStar(c *config.ConfigFactory) *cobra.Command {
	s := &star{configFactory: c}

	cmd := &cobra.Command{
		Use:   "star <project-id>",
		Short: "Mark a project as a starred reference project for the organisation",
		Args:  cobra.ExactArgs(1),
		Run:   s.Run,
	}

	return cmd
}

func (s *star) Run(cmd *cobra.Command, args []string) {
	projectId := args[0]

	err := s.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		WithPath("o/:organisation/projects/" + projectId + "/star").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to star project").WithReason(err).Done()
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "starred")
}
