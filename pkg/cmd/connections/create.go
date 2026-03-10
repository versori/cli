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
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/versori/cli/pkg/browser"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/ulid"
	"github.com/versori/cli/pkg/utils"
)

type credentialFields struct {
	// api key fields
	apiKey string

	// basic auth and oauth2 password
	username string
	password string

	// no auth
	bypass bool

	// oauth2 client
	clientID     string
	clientSecret string
	tokenUrl     string
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
	flags.StringVar(&cr.fields.username, "username", "", "Username for HTTP Basic authentication or OAuth2 password grant type")
	flags.StringVar(&cr.fields.password, "password", "", "Password for HTTP Basic authentication or OAuth2 password grant type")
	flags.BoolVar(&cr.fields.bypass, "bypass", false, "Whether to bypass authentication (if supported by the connection template)")
	flags.StringVar(&cr.fields.clientID, "client-id", "", "OAuth2 client id for use with an oauth2 client connection")
	flags.StringVar(&cr.fields.clientSecret, "client-secret", "", "OAuth2 client secret for use with an oauth2 client connection")
	flags.StringVar(&cr.fields.tokenUrl, "token-url", "", "OAuth2 token URL for use with an oauth2 client connection. Defaults to the token URL defined in the connection template.")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("template-id")

	return cmd
}

func (c *create) Run(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	projectId := c.projectId.GetFlagOrDie(".")
	envId := c.getEnvironment(projectId)

	connectionTemplate := c.getConnectionAuthSchemeConfig(projectId, envId)
	if len(connectionTemplate.AuthSchemeConfigs) == 0 {
		utils.NewExitError().WithMessage("connection template has no auth scheme configs configured").Done()
	}

	// this will cause the cli to exit if there are validation errors
	credentialData := c.createCredentialData(ctx, connectionTemplate.ID.String(), connectionTemplate.AuthSchemeConfigs[0])

	payload := v1.CreateConnectionJSONRequestBody{
		Connection: v1.ConnectionCreate{
			BaseURL: c.baseUrl,
			Credentials: v1.ConnectionCredentialsCreate{
				credentialData,
			},
			Name: c.connectionName,
		},
		EnvironmentSystemID: ulid.MustParse(c.connectionTemplateId),
		ExternalId:          utils.StringOrNil(c.userExternalId),
	}

	resp := v1.Connection{}
	err := c.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		WithPath("o/:organisation/connections").
		Into(&resp).
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to create connection").WithReason(err).Done()
	}

	fmt.Printf("Connection created successfully with ID: %s\n", resp.ID.String())
}

// createCredentialData validates that the flags provided match the auth scheme config.
// If flags are missing, this function will call os.Exit(1) printing the error message
// It returns the credential data for the connection
func (c *create) createCredentialData(ctx context.Context, systemId string, authSchemeConfig v1.AuthSchemeConfig) v1.ConnectionCredential {
	credData := v1.CredentialData{}
	var credType v1.CredentialType

	switch authSchemeConfig.Type {
	case v1.AuthSchemeTypeApiKey:
		if c.fields.apiKey == "" {
			utils.NewExitError().WithMessage("api-key is required for this connection template").Done()
		}

		credType = v1.CredentialTypeString
		credData.String = &v1.CredentialDataString{
			Value: c.fields.apiKey,
		}

	case v1.AuthSchemeTypeBasicAuth:
		if c.fields.username == "" || c.fields.password == "" {
			utils.NewExitError().WithMessage("username and password are required for this connection template").Done()
		}

		credType = v1.CredentialTypeBasicAuth
		credData.BasicAuth = &v1.CredentialDataBasicAuth{
			Password: c.fields.password,
			Username: c.fields.username,
		}

	case v1.AuthSchemeTypeOauth2:
		credData, credType = c.handleOAuth2(ctx, systemId, authSchemeConfig.Oauth2)

	case v1.AuthSchemeTypeNone:
		if !c.fields.bypass {
			utils.NewExitError().WithMessage("bypass is required for this connection template").Done()
		}

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

	default:
		utils.NewExitError().WithMessage("unsupported auth scheme type").WithReason(fmt.Errorf("unsupported auth scheme type: %s", authSchemeConfig.Type)).Done()
	}

	return v1.ConnectionCredential{
		AuthSchemeConfig: &authSchemeConfig,
		Credential: &v1.Credential{
			Data:           &credData,
			OrganisationID: ulid.MustParse(c.configFactory.Context.OrganisationId),
			Type:           credType,
		},
	}
}

func (c *create) handleOAuth2(ctx context.Context, systemId string, authSchemeConfig *v1.AuthSchemeConfigOAuth2) (v1.CredentialData, v1.CredentialType) {
	credData := v1.CredentialData{}
	var credType v1.CredentialType

	switch authSchemeConfig.Grant.Type {
	case v1.AuthSchemeConfigOAuth2GrantTypeAuthorizationCode:
		credData, credType = c.handleOAuth2Code(ctx, systemId, authSchemeConfig)

	case v1.AuthSchemeConfigOAuth2GrantTypeClientCredentials:
		if c.fields.clientID == "" || c.fields.clientSecret == "" {
			utils.NewExitError().WithMessage("client-id and client-secret are required for this connection template").Done()
		}

		credData.Oauth2Client = &v1.CredentialDataOAuth2Client{
			ClientID:     c.fields.clientID,
			ClientSecret: c.fields.clientSecret,
			TokenURL:     utils.DefaultString(c.fields.tokenUrl, authSchemeConfig.TokenURL),
		}

		credType = v1.CredentialTypeOauth2Client

	case v1.AuthSchemeConfigOAuth2GrantTypePassword:
		if c.fields.username == "" || c.fields.password == "" {
			utils.NewExitError().WithMessage("username and password are required for this connection template").Done()
		}

		credData.Oauth2Password = &v1.CredentialDataOAuth2Password{
			Username: c.fields.username,
			Password: c.fields.password,
		}

		credType = v1.CredentialTypeOauth2Password
	default:
		utils.NewExitError().WithMessage("unsupported auth scheme type configuration").WithReason(fmt.Errorf("unsupported oauth2 grant type: %s", authSchemeConfig.Grant.Type)).Done()
	}

	return credData, credType
}

func (c *create) handleOAuth2Code(ctx context.Context, systemId string, authSchemeConfig *v1.AuthSchemeConfigOAuth2) (v1.CredentialData, v1.CredentialType) {
	if authSchemeConfig == nil {
		utils.NewExitError().WithMessage("authSchemeConfig is nil").Done()

		return v1.CredentialData{}, v1.CredentialTypeNone // unreachable because of .Done()
	}

	if authSchemeConfig.Grant.AuthorizationCode.ClientID == nil {
		utils.NewExitError().WithMessage("client-id is required for this connection template").Done()
	}

	closeCh, queryParamCh, redirectUrl := newRedirectServer(ctx)
	defer close(closeCh)

	// initialize connection
	initRequest := v1.InitialiseOAuth2ConnectionRequest{
		AdditionalParams: authSchemeConfig.AdditionalAuthorizeParams,
		RedirectURL:      utils.Ptr(redirectUrl),
		AuthorizeURL:     authSchemeConfig.AuthorizeURL,
		ClientID:         *authSchemeConfig.Grant.AuthorizationCode.ClientID,
		Credential: struct {
			ID             ulid.ULID `json:"id"`
			OrganisationID ulid.ULID `json:"organisationId"`
		}{
			OrganisationID: ulid.MustParse(c.configFactory.Context.OrganisationId),
			ID:             authSchemeConfig.Grant.AuthorizationCode.CredentialID,
		},
		DisableOfflineAccess: true,
		Prompt:               utils.Ptr("consent"),
		Scopes:               convertScopes(authSchemeConfig.Scopes),
	}

	resp := v1.InitialiseOAuth2ConnectionResponse{}
	err := c.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		Into(&resp).
		WithPath("o/:organisation/systems/" + systemId + "/oauth2/initialise").
		JSONBody(initRequest).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to initialize oauth2 connection").WithReason(err).Done()
	}

	err = browser.OpenURL(resp.URL)
	if err != nil {
		utils.NewExitError().WithMessage("failed to open browser").WithReason(err).Done()
	}

	query := map[string][]string{}
	select {
	case query = <-queryParamCh:
	case <-time.After(15 * time.Minute):
		utils.NewExitError().WithMessage("timed out waiting for redirect server to receive request").Done()
	case <-ctx.Done():
		utils.NewExitError().WithMessage("cancelled waiting for redirect server to receive request").Done()
	}

	code := getValue("code", query)
	state := getValue("state", query)

	switch {
	case code == "":
		utils.NewExitError().WithMessage("code is required in the redirect url for oauth2 connections to work").Done()

	case state == "":
		utils.NewExitError().WithMessage("state is required in the redirect url for oauth2 connections to work").Done()
	}

	return v1.CredentialData{
		Oauth2Code: &v1.CredentialDataOAuth2Code{
			Code:             code,
			State:            state,
			AdditionalParams: authSchemeConfig.AdditionalTokenParams,
			RedirectURL:      utils.Ptr(redirectUrl),
		},
	}, v1.CredentialTypeOauth2Code
}

func (c *create) getEnvironment(projectId string) string {
	project := v1.Project{}
	err := c.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
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

	return env.ID.String()
}

func (c *create) getConnectionAuthSchemeConfig(projectId, envId string) v1.ConnectionTemplate {
	connTemplatesPage := v1.EnvironmentSystemPage{}
	err := c.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&connTemplatesPage).
		WithPath("o/:organisation/projects/"+projectId+"/connection-templates").
		WithQueryParam("env_id", envId).
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

	return connTemplate
}

func convertScopes(scopes []v1.OAuth2Scope) []string {
	converted := make([]string, len(scopes))
	for i, s := range scopes {
		converted[i] = s.Name
	}

	return converted
}

// newRedirectServer starts a new HTTP server on a random port and returns a channel to close the server, a channel to receive the query parameters and the redirect URL.
// It is the callers responsibility to close the close channel when the server is no longer needed.
// The query channel will be closed after the first request is received.
func newRedirectServer(ctx context.Context) (chan struct{}, chan map[string][]string, string) {
	randomPort := 62168 // some oauth2 providers need the port to be fixed
	queryParamCh := make(chan map[string][]string)
	closeChannel := make(chan struct{})
	redirectUrl := fmt.Sprintf("http://127.0.0.1:%d/oauth/callback", randomPort)

	handler := http.NewServeMux()
	handler.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		queryParamCh <- r.URL.Query()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(oauth2Screen))
	})

	s := http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", randomPort),
		Handler: handler,
	}

	go func() {
		select {
		case <-ctx.Done():
			err := s.Shutdown(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to shutdown redirect server: %v\n", err)
			}
		case <-closeChannel:
			err := s.Shutdown(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to shutdown redirect server: %v\n", err)
			}
		}

		close(queryParamCh)
	}()

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server ListenAndServe: %v\n", err)
		}
	}()

	return closeChannel, queryParamCh, redirectUrl
}

func getValue(key string, m map[string][]string) string {
	if m == nil {
		return ""
	}

	arr := m[key]
	if len(arr) == 0 {
		return ""
	}

	return arr[0]
}
