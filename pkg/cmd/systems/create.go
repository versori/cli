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
	"net/http"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/utils"
)

type create struct {
	configFactory   *config.ConfigFactory
	name            string
	domain          string
	templateBaseUrl string
}

func NewCreate(c *config.ConfigFactory) *cobra.Command {
	cr := &create{configFactory: c}

	cmd := &cobra.Command{
		Use:   "create --name <name> --domain <domain> --template-base-url <url>",
		Short: "Create a new system in the current organisation",
		Run:   cr.Run,
	}

	flags := cmd.Flags()
	flags.StringVar(&cr.name, "name", "", "Name of the system")
	flags.StringVar(&cr.domain, "domain", "", "Domain of the system")
	flags.StringVar(&cr.templateBaseUrl, "template-base-url", "", "Template base URL for the system")

	return cmd
}

func (c *create) Run(_ *cobra.Command, _ []string) {
	payload := v1.CreateSystemJSONRequestBody{
		Name:              c.name,
		Domain:            c.domain,
		TemplateBaseUrl:   c.templateBaseUrl,
		AuthSchemeConfigs: nil,
	}

	if c.name == "" {
		editor := elements.NewEditor("Enter system name:", false)
		err := editor.Edit(&payload.Name)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read system name").WithReason(err).Done()
		}
	}

	if c.domain == "" {
		editor := elements.NewEditor("Enter system domain:", false, elements.WithValidation(utils.IsValidURL))
		err := editor.Edit(&payload.Domain)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read system domain").WithReason(err).Done()
		}
	}

	if c.templateBaseUrl == "" {
		editor := elements.NewEditor("Enter system template Base URL:", false, elements.WithValidation(utils.IsValidURL))
		err := editor.Edit(&payload.TemplateBaseUrl)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read system template base URL").WithReason(err).Done()
		}
	}

	resp := v1.System{}
	err := c.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		WithPath("o/:organisation/systems").
		Into(&resp).
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to create system").WithReason(err).Done()
	}

	pi := printableSystem{
		Name:              resp.Name,
		Id:                resp.ID.String(),
		Domain:            resp.Domain,
		TemplateBaseUrl:   resp.TemplateBaseUrl,
		AuthSchemeConfigs: resp.AuthSchemeConfigs,
	}

	c.configFactory.Print(pi)
}
