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

package systems

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type bootstrapRequest struct {
	OrganisationID  string         `json:"organisation_id"`
	ProjectID       string         `json:"project_id"`
	ResearchContext string         `json:"research_context"`
	SystemOverrides map[string]any `json:"system_overrides,omitempty"`
}

type bootstrapResponse struct {
	RegisteredSystems []registeredSystem `json:"registered_systems"`
	FailedSystems     []failedSystem     `json:"failed_systems"`
	PendingQuestions  []any              `json:"pending_questions"`
	Message           string             `json:"message"`
}

type registeredSystem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Domain          string `json:"domain"`
	BaseURL         string `json:"base_url"`
	LinkedToProject bool   `json:"linked_to_project"`
	Created         bool   `json:"created"`
	Updated         bool   `json:"updated"`
}

type failedSystem struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

type bootstrap struct {
	configFactory   *config.ConfigFactory
	projectId       flags.ProjectId
	file            string
	systemOverrides string
}

func NewBootstrap(c *config.ConfigFactory) *cobra.Command {
	b := &bootstrap{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "bootstrap --file <path>",
		Short: "Create systems from research context",
		Run:   b.Run,
	}

	f := cmd.Flags()
	b.projectId.SetFlag(f)
	f.StringVarP(&b.file, "file", "f", "", "Path to a file containing research context (required)")
	f.StringVar(&b.systemOverrides, "system-overrides", "", `JSON object of per-system overrides (e.g. '{"Stripe": {"base_url": "..."}}')`)

	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func (b *bootstrap) Run(cmd *cobra.Command, args []string) {
	projectId := b.projectId.GetFlagOrDie(".")
	orgId := b.configFactory.Context.OrganisationId

	data, err := os.ReadFile(b.file)
	if err != nil {
		utils.NewExitError().WithMessage("failed to read file").WithReason(err).Done()
	}

	body := bootstrapRequest{
		OrganisationID:  orgId,
		ProjectID:       projectId,
		ResearchContext: string(data),
	}

	if b.systemOverrides != "" {
		var overrides map[string]any
		if err := json.Unmarshal([]byte(b.systemOverrides), &overrides); err != nil {
			utils.NewExitError().WithMessage("invalid --system-overrides JSON").WithReason(err).Done()
		}
		body.SystemOverrides = overrides
	}

	var resp bootstrapResponse

	err = b.configFactory.
		NewAIRequest().
		WithMethod(http.MethodPost).
		JSONBody(body).
		Into(&resp).
		WithPath("assistant/v2/connectors/bootstrap").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to bootstrap systems").WithReason(err).Done()
	}

	fmt.Println(resp.Message)

	if len(resp.RegisteredSystems) > 0 {
		fmt.Println("\nRegistered systems:")
		for _, s := range resp.RegisteredSystems {
			status := ""
			if s.Created {
				status = " (created)"
			} else if s.Updated {
				status = " (updated)"
			}
			fmt.Printf("  - %s [%s] %s%s\n", s.Name, s.ID, s.BaseURL, status)
		}
	}

	if len(resp.FailedSystems) > 0 {
		fmt.Println("\nFailed systems:")
		for _, s := range resp.FailedSystems {
			fmt.Printf("  - %s: %s\n", s.Name, s.Error)
		}
	}
}
