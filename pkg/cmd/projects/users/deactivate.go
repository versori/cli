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
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type deactivate struct {
	configFactory   *config.ConfigFactory
	projectId       flags.ProjectId
	environmentName string
	userExternalId  string
}

func NewDeactivate(c *config.ConfigFactory) *cobra.Command {
	d := &deactivate{configFactory: c}

	cmd := &cobra.Command{
		Use:   "deactivate --project <project-id> --environment <environment-name> --external-id <user-external-id>",
		Short: "Deactivate an end-user on a project environment (deletes the activation)",
		Long: `Deactivate an end-user on a project environment by deleting the activation record.
The end-user themselves and any embedded connections are NOT deleted — only the link between
them and the environment. To re-activate the same end-user, run 'versori projects users activate'
again.`,
		Run: d.Run,
	}

	f := cmd.Flags()
	d.projectId.SetFlag(f)
	f.StringVar(&d.environmentName, "environment", "", "The environment name within the project")
	f.StringVarP(&d.userExternalId, "external-id", "e", "", "External ID of the end-user to deactivate")

	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("external-id")

	return cmd
}

func (d *deactivate) Run(_ *cobra.Command, _ []string) {
	projectId := d.projectId.GetFlagOrDie(".")

	envId := resolveEnvironmentID(d.configFactory, projectId, d.environmentName)
	activationId := resolveActivationID(d.configFactory, envId, d.userExternalId)

	err := d.configFactory.
		NewRequest().
		WithMethod(http.MethodDelete).
		WithPath("o/:organisation/environments/" + envId + "/activations/" + activationId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to deactivate user").WithReason(err).Done()
	}

	fmt.Printf("Activation deleted for end-user %q on environment %q\n", d.userExternalId, d.environmentName)
}
