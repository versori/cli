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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type setVariable struct {
	configFactory   *config.ConfigFactory
	projectId       flags.ProjectId
	environmentName string
	userExternalId  string
	name            string
	value           string
}

type setVariableBody struct {
	Value any `json:"value"`
}

func NewSetVariable(c *config.ConfigFactory) *cobra.Command {
	s := &setVariable{configFactory: c}

	cmd := &cobra.Command{
		Use:   "set-variable --project <project-id> --environment <environment-name> --external-id <user-external-id> --name <variable-name> --value <value>",
		Short: "Set a single dynamic variable on an end-user's activation",
		Long: `Set a single dynamic variable on an end-user's activation. The variable name must be
declared in the project's DynamicVariablesSchema first (manage it via 'versori projects variables
set/patch'); unknown keys are rejected by the platform.

The --value flag is parsed as JSON when valid (so '42', 'true', '"hello"', '{"a":1}' all work);
otherwise it is treated as a raw string. Variable updates take effect immediately at runtime —
no redeploy required.`,
		Run: s.Run,
	}

	f := cmd.Flags()
	s.projectId.SetFlag(f)
	f.StringVar(&s.environmentName, "environment", "", "The environment name within the project")
	f.StringVarP(&s.userExternalId, "external-id", "e", "", "External ID of the end-user")
	f.StringVarP(&s.name, "name", "n", "", "Name of the variable to set")
	f.StringVar(&s.value, "value", "", "Value for the variable (parsed as JSON when valid, else treated as a string)")

	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("external-id")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("value")

	return cmd
}

func (s *setVariable) Run(_ *cobra.Command, _ []string) {
	projectId := s.projectId.GetFlagOrDie(".")

	envId := resolveEnvironmentID(s.configFactory, projectId, s.environmentName)
	activationId := resolveActivationID(s.configFactory, envId, s.userExternalId)

	var parsed any
	if err := json.Unmarshal([]byte(s.value), &parsed); err != nil {
		parsed = s.value
	}

	body := setVariableBody{Value: parsed}

	err := s.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		WithPath("o/:organisation/environments/" + envId + "/activations/" + activationId + "/variables/" + s.name).
		JSONBody(body).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to set dynamic variable").WithReason(err).Done()
	}

	fmt.Printf("Variable %q updated on activation for end-user %q\n", s.name, s.userExternalId)
}
