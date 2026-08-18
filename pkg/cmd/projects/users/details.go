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
	"net/http"

	"github.com/spf13/cobra"
	"github.com/versori/cli/pkg/ulid"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

func init() {
	utils.RegisterResource(bigActivation{}, []string{"ID", "User.DisplayName", "User.ExternalID", "Connections.Name", "DynamicVariables"})
}

type bigActivation struct {
	Connections      []v1.Connection     `json:"connections"`
	DynamicVariables v1.DynamicVariables `json:"dynamicVariables"`
	ID               ulid.ULID           `json:"id"`
	User             v1.EndUser          `json:"user"`
}

type detailsActivation struct {
	configFactory   *config.ConfigFactory
	projectId       flags.ProjectId
	environmentName string
	user            string
}

func NewDetailsActivation(c *config.ConfigFactory) *cobra.Command {
	l := &detailsActivation{configFactory: c}

	cmd := &cobra.Command{
		Use:   "details --project <project-id> --environment <environment-name> --external-id <user-id>",
		Short: "Get the details for an individual end user of a project",
		Run:   l.Run,
	}

	flags := cmd.Flags()
	l.projectId.SetFlag(flags)
	flags.StringVar(&l.environmentName, "environment", "", "The environment name within the project")
	flags.StringVar(&l.user, "external-id", "", "The external ID of the user")

	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("external-id")

	return cmd
}

func (l *detailsActivation) Run(cmd *cobra.Command, args []string) {
	projectId := l.projectId.GetFlagOrDie(".")

	// Fetch project to resolve environment ID from name
	project := v1.Project{}
	err := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&project).
		WithPath("o/:organisation/projects/" + projectId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get project").WithReason(err).Done()
	}

	// Resolve environment ID by name
	envId := ""
	for _, e := range project.Environments {
		if e.Name == l.environmentName {
			envId = e.ID.String()

			break
		}
	}
	if envId == "" {
		utils.NewExitError().WithMessage("environment [" + l.environmentName + "] not found in project").Done()
	}

	// Call activations endpoint for the environment
	var resp []v1.Activation
	err = l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/environments/" + envId + "/activations").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list activations").WithReason(err).Done()
	}

	// find activationID based on external id for the user
	activationId := ""
	for _, a := range resp {
		if a.User.ExternalID == l.user {
			activationId = a.ID.String()

			break
		}
	}

	if activationId == "" {
		utils.NewExitError().WithMessage("No user found for external ID: " + l.user).Done()
	}

	var body []byte
	err = l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&body).
		WithPath("o/:organisation/environments/" + envId + "/activations/" + activationId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get activation").WithReason(err).Done()
	}

	ba, err := parseActivationDetails(body)
	if err != nil {
		utils.NewExitError().WithMessage("failed to decode activation").WithReason(err).Done()
	}

	fallback := []v1.Connection{}
	if len(ba.Connections) == 0 {
		if userULID := endUserULID(l.configFactory, ba.User); userULID != "" {
			fallback = listConnectionsForEndUser(l.configFactory, userULID)
		}
	}

	ba.Connections = hydrateConnections(l.configFactory, resolveDetailConnections(ba.Connections, fallback))

	l.configFactory.Print(ba)
}

// parseActivationDetails reads a GET-activation body without requiring the
// generated Connection/Credential union to decode. Credential bags on embedded
// connections are often sparse or use scheme types the CLI types don't model;
// failing the whole details command on that left VS Code with an empty
// connections list even when the activation payload named them.
func parseActivationDetails(raw []byte) (bigActivation, error) {
	var dto struct {
		Connections      []json.RawMessage   `json:"connections"`
		DynamicVariables v1.DynamicVariables `json:"dynamicVariables"`
		ID               ulid.ULID           `json:"id"`
		User             v1.EndUser          `json:"user"`
	}
	if err := json.Unmarshal(raw, &dto); err != nil {
		return bigActivation{}, err
	}

	ba := bigActivation{
		Connections:      make([]v1.Connection, 0, len(dto.Connections)),
		DynamicVariables: dto.DynamicVariables,
		ID:               dto.ID,
		User:             dto.User,
	}

	for _, rawConn := range dto.Connections {
		var ident struct {
			BaseURL             string `json:"baseUrl"`
			EnvironmentSystemID string `json:"connectionTemplateId"`
			ID                  string `json:"id"`
			Name                string `json:"name"`
			SystemID            string `json:"systemId"`
		}
		if err := json.Unmarshal(rawConn, &ident); err != nil {
			continue
		}

		conn := v1.Connection{
			BaseURL: ident.BaseURL,
			Name:    ident.Name,
		}
		if id, err := ulid.Parse(ident.ID); err == nil {
			conn.ID = id
		}
		if sid, err := ulid.Parse(ident.SystemID); err == nil {
			conn.SystemID = sid
		}
		if tid, err := ulid.Parse(ident.EnvironmentSystemID); err == nil {
			conn.EnvironmentSystemID = tid
		}
		ba.Connections = append(ba.Connections, conn)
	}

	return ba, nil
}

// resolveDetailConnections prefers the connections bound on the activation.
// GetActivation treats a failed connection load as non-fatal and omits the
// field; the UI then lists the end-user's embedded connections instead. Mirror
// that fallback so `activations details` (and the VS Code panel that shells
// out to it) does not render "no connections" for an activated external user.
func resolveDetailConnections(bound, fallback []v1.Connection) []v1.Connection {
	if len(bound) > 0 {
		return bound
	}
	return fallback
}

func endUserULID(cf *config.ConfigFactory, user v1.EndUser) string {
	if !user.ID.IsZero() {
		return user.ID.String()
	}
	if user.ExternalID == "" {
		return ""
	}

	resp := v1.EndUserPage{}
	err := cf.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/users").
		Do()
	if err != nil {
		return ""
	}
	for _, u := range resp.Users {
		if u.ExternalID == user.ExternalID {
			return u.ID.String()
		}
	}
	return ""
}

func listConnectionsForEndUser(cf *config.ConfigFactory, userULID string) []v1.Connection {
	resp := v1.ConnectionPage{}
	err := cf.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/connections").
		WithQueryParam("end_user_id", userULID).
		Do()
	if err != nil {
		return nil
	}
	return resp.Items
}

// hydrateConnections best-effort fills each stub from GET /connections/{id}.
// A failed hydrate keeps the stub — aborting used to hide every connection
// when a single embedded credential payload would not decode.
func hydrateConnections(cf *config.ConfigFactory, stubs []v1.Connection) []v1.Connection {
	out := make([]v1.Connection, 0, len(stubs))
	for _, stub := range stubs {
		if stub.ID.IsZero() {
			out = append(out, stub)
			continue
		}

		full := v1.Connection{}
		err := cf.
			NewRequest().
			WithMethod(http.MethodGet).
			Into(&full).
			WithPath("o/:organisation/connections/" + stub.ID.String()).
			Do()
		if err != nil || full.ID.IsZero() {
			out = append(out, stub)
			continue
		}
		out = append(out, full)
	}
	return out
}
