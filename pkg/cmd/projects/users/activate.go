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
	"os"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/ulid"
	"github.com/versori/cli/pkg/utils"
)

type activate struct {
	configFactory   *config.ConfigFactory
	projectId       flags.ProjectId
	environmentName string
	userExternalId  string
	connectionPairs []string
	variablePairs   []string
	variablesFile   string
}

func NewActivate(c *config.ConfigFactory) *cobra.Command {
	a := &activate{configFactory: c}

	cmd := &cobra.Command{
		Use:   "activate --project <project-id> --environment <environment-name> --external-id <user-external-id> --connection <system-id>=<connection-id> [--variable key=value]... [--variables-file <path>]",
		Short: "Activate an end-user on a project environment",
		Long: `Activate an end-user on a project environment. The activation links an end-user to a
specific connection per environment system, plus an optional bag of dynamic variables that
workflow code reads via ctx.activation.getVariable('<key>').

The number of --connection flags must equal the number of environment systems for the target
environment (use 'versori projects systems list --project <id> --environment <env>' to enumerate them).

Dynamic variables can be supplied inline via repeatable --variable key=value flags, or in bulk
from a JSON file via --variables-file. Variables are validated against the project's
DynamicVariablesSchema (manage it with 'versori projects variables set/patch'); unknown keys fail.

End-users themselves are created with 'versori users create -e <external-id> -n <display-name>'.
Connections are created with 'versori connections create' (use --external-id <user> for embedded
per-end-user connections). Once both exist, this command links them together.`,
		Run: a.Run,
	}

	f := cmd.Flags()
	a.projectId.SetFlag(f)
	f.StringVar(&a.environmentName, "environment", "", "The environment name within the project")
	f.StringVarP(&a.userExternalId, "external-id", "e", "", "External ID of the end-user to activate")
	f.StringSliceVar(&a.connectionPairs, "connection", nil, "Connection pair in the form <system-template-id>=<connection-id> (repeatable; one per environment system)")
	f.StringSliceVar(&a.variablePairs, "variable", nil, "Dynamic variable in the form key=value (repeatable). Values are parsed as JSON when valid, else treated as strings.")
	f.StringVar(&a.variablesFile, "variables-file", "", "Path to a JSON file containing a flat object of dynamic variables (merged with --variable; --variable wins on conflicts)")

	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("external-id")
	_ = cmd.MarkFlagRequired("connection")

	return cmd
}

func (a *activate) Run(_ *cobra.Command, _ []string) {
	projectId := a.projectId.GetFlagOrDie(".")

	envId := resolveEnvironmentID(a.configFactory, projectId, a.environmentName)

	connections, err := parseConnectionPairs(a.connectionPairs)
	if err != nil {
		utils.NewExitError().WithMessage("invalid --connection flag").WithReason(err).Done()
	}

	variables, err := mergeVariables(a.variablesFile, a.variablePairs)
	if err != nil {
		utils.NewExitError().WithMessage("invalid dynamic variables").WithReason(err).Done()
	}

	payload := v1.ActivationCreate{
		Connections:      connections,
		DynamicVariables: variables,
		EnvironmentID:    ulid.MustParse(envId),
		UserID:           a.userExternalId,
	}

	resp := v1.Activation{}
	err = a.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		WithPath("o/:organisation/activations").
		Into(&resp).
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to activate user").WithReason(err).Done()
	}

	fmt.Printf("Activation created successfully with ID: %s\n", resp.ID.String())
}

// resolveEnvironmentID fetches a project and returns the ID of the environment with the given name.
// Exits with an error if the environment is not found.
func resolveEnvironmentID(cf *config.ConfigFactory, projectId, envName string) string {
	project := v1.Project{}
	err := cf.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&project).
		WithPath("o/:organisation/projects/" + projectId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get project").WithReason(err).Done()
	}

	for _, e := range project.Environments {
		if e.Name == envName {
			return e.ID.String()
		}
	}

	utils.NewExitError().WithMessage("environment [" + envName + "] not found in project").Done()

	return "" // unreachable
}

// parseConnectionPairs converts repeated --connection <system-template-id>=<connection-id> flags
// into the ActivationCreate.Connections slice shape that the platform expects.
func parseConnectionPairs(pairs []string) ([]struct {
	Connection           *v1.ConnectionCreate    `json:"connection,omitempty"`
	EnvironmentSystemID  v1.EnvironmentSystemID  `json:"connectionTemplateId"`
	ExistingConnectionID *ulid.ULID              `json:"existingConnectionId,omitempty"`
}, error) {
	out := make([]struct {
		Connection           *v1.ConnectionCreate    `json:"connection,omitempty"`
		EnvironmentSystemID  v1.EnvironmentSystemID  `json:"connectionTemplateId"`
		ExistingConnectionID *ulid.ULID              `json:"existingConnectionId,omitempty"`
	}, 0, len(pairs))

	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("expected <system-template-id>=<connection-id>, got %q", p)
		}
		tplID, err := ulid.Parse(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid system-template-id %q: %w", parts[0], err)
		}
		connID, err := ulid.Parse(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid connection-id %q: %w", parts[1], err)
		}
		out = append(out, struct {
			Connection           *v1.ConnectionCreate    `json:"connection,omitempty"`
			EnvironmentSystemID  v1.EnvironmentSystemID  `json:"connectionTemplateId"`
			ExistingConnectionID *ulid.ULID              `json:"existingConnectionId,omitempty"`
		}{
			EnvironmentSystemID:  tplID,
			ExistingConnectionID: &connID,
		})
	}

	return out, nil
}

// mergeVariables combines an optional JSON file with --variable key=value flags.
// --variable wins on conflicts. Values parse as JSON when valid, else are treated as strings.
func mergeVariables(filePath string, pairs []string) (v1.DynamicVariables, error) {
	out := v1.DynamicVariables{}

	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read variables file: %w", err)
		}
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, fmt.Errorf("variables file is not valid JSON object: %w", err)
		}
	}

	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("expected key=value, got %q", p)
		}
		key, raw := parts[0], parts[1]

		var parsed any
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			out[key] = parsed
		} else {
			out[key] = raw
		}
	}

	if len(out) == 0 {
		return nil, nil
	}

	return out, nil
}
