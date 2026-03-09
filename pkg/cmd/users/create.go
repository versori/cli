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

package users

import (
	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/utils"
)

type createUser struct {
	configFactory *config.ConfigFactory
	displayName   string
	externalId    string
}

func NewCreate(c *config.ConfigFactory) *cobra.Command {
	cu := &createUser{configFactory: c}

	cmd := &cobra.Command{
		Use:   "create --display-name <name> --external-id <id>",
		Short: "Create a new user in the current organisation",
		Run:   cu.Run,
	}

	flags := cmd.Flags()
	flags.StringVarP(&cu.displayName, "display-name", "n", "", "Display name of the user")
	flags.StringVarP(&cu.externalId, "external-id", "e", "", "External ID of the user")

	return cmd
}

func (cu *createUser) Run(_ *cobra.Command, _ []string) {
	if cu.displayName == "" {
		dnEditor := elements.NewEditor("Enter display name:", false)
		err := dnEditor.Edit(&cu.displayName)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read display name").WithReason(err).Done()
		}
	}

	if cu.externalId == "" {
		eidEditor := elements.NewEditor("Enter external ID:", false)
		err := eidEditor.Edit(&cu.externalId)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read external ID").WithReason(err).Done()
		}
	}

	req := v1.CreateEndUserJSONRequestBody{
		DisplayName: cu.displayName,
		ExternalId:  cu.externalId,
	}

	err := cu.configFactory.
		NewRequest().
		WithMethod("POST").
		WithPath("o/:organisation/users").
		JSONBody(req).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to create user").WithReason(err).Done()
	}
}
