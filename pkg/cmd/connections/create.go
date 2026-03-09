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

package connections

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/ulid"
	"github.com/versori/cli/pkg/utils"
)

type credentialFields struct {
	apiKey   string
	username string
	password string
	bypass   bool
}

type create struct {
	configFactory        *config.ConfigFactory
	projectId            flags.ProjectId
	envName              string
	connectionTemplateId string
	userExternalId       string
	connectionName       string
	baseUrl              string
	fields               credentialFields
}

func NewCreate(c *config.ConfigFactory) *cobra.Command {
	cr := &create{configFactory: c}

	cmd := &cobra.Command{
		Use:   "create --project <project-id> --environment <environment-name> --name <name> --template-id <template-id> [--external-id <external-id>] [--base-url <base-url>] [--<credential-field> <value>]...",
		Short: "Create a new connection to a connection template",
		Long: `Create a new connection to a connection template. If an end user's external ID is provided,
The connection will be created as a dynamic connection for that end user, otherwise it will be created as a static connection.
If a base URL is not provided, it will default to the system's base URL defined in the connection template.`,
		Run: cr.Run,
	}

	flags := cmd.Flags()

	cr.projectId.SetFlag(flags)

	flags.StringVar(&cr.envName, "environment", "", "The environment name within the project")
	flags.StringVar(&cr.connectionName, "name", "", "Name of the connection")
	flags.StringVar(&cr.connectionTemplateId, "template-id", "", "ID of the connection template to connect to")
	flags.StringVar(&cr.userExternalId, "external-id", "", "External ID of the end user for the connection, if not provided the connection will be created as a static connection")
	flags.StringVar(&cr.baseUrl, "base-url", "", "Base URL for the connection, if not provided it will default to the systems base URL")

	// Credential fields
	flags.StringVar(&cr.fields.apiKey, "api-key", "", "API key for authentication")
	flags.StringVar(&cr.fields.username, "username", "", "Username for HTTP Basic authentication")
	flags.StringVar(&cr.fields.password, "password", "", "Password for HTTP Basic authentication")
	flags.BoolVar(&cr.fields.bypass, "bypass", false, "Whether to bypass authentication (if supported by the connection template)")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("template-id")

	return cmd
}

// validateCredentialFields validates that only one of the following is set:
// - api-key
// - bypass
// - both username AND password
func (c *create) validateCredentialFields() error {
	numSet := 0

	if c.fields.apiKey != "" {
		numSet++
	}
	if c.fields.bypass {
		numSet++
	}
	if c.fields.username != "" || c.fields.password != "" {
		// Both username and password must be set together
		if (c.fields.username == "") || (c.fields.password == "") {
			return fmt.Errorf("both username and password must be provided together")
		}
		numSet++
	}

	if numSet == 0 {
		return fmt.Errorf("must provide one of: --api-key, --bypass, or (--username AND --password)")
	}
	if numSet > 1 {
		return fmt.Errorf("only one of the following can be set: --api-key, --bypass, or (--username AND --password)")
	}

	return nil
}

func (c *create) Run(_ *cobra.Command, _ []string) {
	currentDir, err := os.Getwd()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get current directory").WithReason(err).Done()
	}

	projectId := c.projectId.GetFlagOrDie(currentDir)

	// Validate credential fields
	if err := c.validateCredentialFields(); err != nil {
		utils.NewExitError().WithMessage("invalid credential fields").WithReason(err).Done()
	}

	// first get the authscheme config from the API for the provided templateID
	project := v1.Project{}
	err = c.configFactory.
		NewRequest().
		WithMethod("GET").
		Into(&project).
		WithPath("o/:organisation/projects/" + projectId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get project").WithReason(err).Done()
	}

	var env v1.ProjectEnvironment
	for _, e := range project.Environments {
		if e.Name == c.envName {
			env = e

			break
		}
	}

	if env.ID.IsZero() {
		utils.NewExitError().WithMessage("environment [" + c.envName + "] not found in project").Done()
	}

	connTemplatesPage := v1.EnvironmentSystemPage{}
	err = c.configFactory.
		NewRequest().
		WithMethod("GET").
		Into(&connTemplatesPage).
		WithPath("o/:organisation/projects/"+projectId+"/connection-templates").
		WithQueryParam("env_id", env.ID.String()).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get connection templates for environment").WithReason(err).Done()
	}

	connTemplate := v1.ConnectionTemplate{}
	for _, ct := range connTemplatesPage.Items {
		if ct.EnvironmentSystemID.String() == c.connectionTemplateId {
			connTemplate = ct

			break
		}
	}

	if connTemplate.ID.IsZero() {
		utils.NewExitError().WithMessage("connection template not found in environment").Done()
	}

	payload := v1.CreateConnectionJSONRequestBody{
		Connection: v1.ConnectionCreate{
			BaseURL: c.baseUrl,
			Credentials: v1.ConnectionCredentialsCreate{
				c.createConnectionCredentials(connTemplate.AuthSchemeConfigs[0]),
			},
			Name: c.connectionName,
		},
		EnvironmentSystemID: ulid.MustParse(c.connectionTemplateId),
		ExternalId:          utils.StringOrNil(c.userExternalId),
	}

	resp := v1.Connection{}
	err = c.configFactory.
		NewRequest().
		WithMethod("POST").
		WithPath("o/:organisation/connections").
		Into(&resp).
		JSONBody(payload).
		Do()
	if err != nil {
		fmt.Println(err.Error())
		utils.NewExitError().WithMessage("failed to create connection").WithReason(err).Done()
	}

	fmt.Printf("Connection created successfully with ID: %s\n", resp.ID.String())
}

func (c *create) createConnectionCredentials(asc v1.AuthSchemeConfig) v1.ConnectionCredential {
	credData := v1.CredentialData{}
	var credType v1.CredentialType

	// very simple right now, we just go through the struct fields and check.
	// TODO: clean this up maybe do reflection stuff idk

	switch {
	case c.fields.bypass:
		credType = v1.CredentialTypeNone

		return v1.ConnectionCredential{
			AuthSchemeConfig: &v1.AuthSchemeConfig{
				None: &v1.AuthSchemeConfigNone{},
				Type: v1.AuthSchemeTypeNone,
			},
			Credential: &v1.Credential{
				OrganisationID: ulid.MustParse(c.configFactory.Context.OrganisationId),
				Type:           credType,
			},
		}
	case c.fields.apiKey != "":
		credType = v1.CredentialTypeString

		if asc.Type != v1.AuthSchemeTypeApiKey {
			utils.NewExitError().WithMessage("provided API key credential fields, but the connection template does not support API key authentication").Done()
		}

		credData.String = &v1.CredentialDataString{
			Value: c.fields.apiKey,
		}
	case c.fields.username != "" && c.fields.password != "":
		credType = v1.CredentialTypeBasicAuth

		if asc.Type != v1.AuthSchemeTypeApiKey {
			utils.NewExitError().WithMessage("provided API key credential fields, but the connection template does not support API key authentication").Done()
		}

		credData.BasicAuth = &v1.CredentialDataBasicAuth{
			Password: c.fields.password,
			Username: c.fields.username,
		}
	default:
		// hopefully validation should catch this before we get here, but just in case
		utils.NewExitError().WithMessage("invalid credential fields").Done()
	}

	return v1.ConnectionCredential{
		AuthSchemeConfig: &asc,
		Credential: &v1.Credential{
			Data:           &credData,
			OrganisationID: ulid.MustParse(c.configFactory.Context.OrganisationId),
			Type:           credType,
		},
	}
}
