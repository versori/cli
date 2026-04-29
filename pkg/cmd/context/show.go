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

package context

import (
	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

// ContextSummary is the public, redacted view of a context — no JWT.
type ContextSummary struct {
	Name              string `json:"name" yaml:"name"`
	OrganisationId    string `json:"organisationId" yaml:"organisation_id"`
	URLOverwrite      string `json:"urlOverwrite,omitempty" yaml:"url_overwrite,omitempty"`
	DisableReferences bool   `json:"disableReferences" yaml:"disable_references"`
}

type show struct {
	configFactory *config.ConfigFactory
}

func init() {
	utils.RegisterResource(ContextSummary{}, []string{"Name", "OrganisationId", "DisableReferences"})
}

func NewShow(c *config.ConfigFactory) *cobra.Command {
	s := &show{configFactory: c}

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the active context (without secrets).",
		Run:   s.Run,
	}

	return cmd
}

func (s *show) Run(cmd *cobra.Command, args []string) {
	s.configFactory.LoadConfigAndContext()

	ctx := s.configFactory.Context

	summary := ContextSummary{
		Name:              ctx.Name,
		OrganisationId:    ctx.OrganisationId,
		URLOverwrite:      ctx.URLOverwrite,
		DisableReferences: ctx.DisableReferences,
	}

	s.configFactory.Print(summary)
}
