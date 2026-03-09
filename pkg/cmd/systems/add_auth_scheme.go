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
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type addAuthScheme struct {
	configFactory *config.ConfigFactory

	// common flags
	systemID    string
	schemeType  string
	name        string
	description string

	// api-key flags
	apiKeyName string
	apiKeyIn   string

	// oauth2 flags
	oauth2TokenURL      string
	oauth2AuthorizeURL  string
	oauth2GrantType     string
	oauth2ClientID      string
	oauth2ClientSecret  string
	oauth2Scopes        []string
	oauth2DefaultScopes []string

	// hmac flags
	hmacName         string
	hmacIn           string
	hmacAlgorithm    string
	hmacEncoding     string
	hmacDigestInputs []string
}

func NewAddAuthScheme(c *config.ConfigFactory) *cobra.Command {
	a := &addAuthScheme{configFactory: c}

	cmd := &cobra.Command{
		Use:   "add-auth-scheme --system-id <system-id> --type <type> --name <name>",
		Short: "Add or update an auth scheme config for a system",
		Long: `Adds or updates an auth scheme config for the given system.
If the auth scheme name already exists on the system it will be updated, otherwise it will be created.

Supported types: none, api-key, basic-auth, oauth2, hmac, certificate`,
		Run: a.Run,
	}

	flags := cmd.Flags()

	// common
	flags.StringVar(&a.systemID, "system-id", "", "ID of the system to add the auth scheme to (required)")
	flags.StringVar(&a.schemeType, "type", "", "Auth scheme type (none, api-key, basic-auth, oauth2, hmac, certificate)")
	flags.StringVar(&a.name, "name", "", "Name of the auth scheme config")
	flags.StringVar(&a.description, "description", "", "Human-readable description for this auth scheme config")

	// api-key
	flags.StringVar(&a.apiKeyName, "api-key.name", "", "Header/query/cookie name for API key (required when type=api-key)")
	flags.StringVar(&a.apiKeyIn, "api-key.in", "", "Where to send the API key: header, query or cookie (required when type=api-key)")

	// oauth2
	flags.StringVar(&a.oauth2TokenURL, "oauth2.token-url", "", "OAuth2 token URL (required when type=oauth2)")
	flags.StringVar(&a.oauth2AuthorizeURL, "oauth2.authorize-url", "", "OAuth2 authorization URL")
	flags.StringVar(&a.oauth2GrantType, "oauth2.grant-type", "authorizationCode", "OAuth2 grant type: authorizationCode, clientCredentials, password")
	flags.StringVar(&a.oauth2ClientID, "oauth2.client-id", "", "OAuth2 client ID")
	flags.StringVar(&a.oauth2ClientSecret, "oauth2.client-secret", "", "OAuth2 client secret")
	flags.StringArrayVar(&a.oauth2Scopes, "oauth2.scope", nil, "OAuth2 scope in name=description format, e.g. read=Read access (repeatable)")
	flags.StringArrayVar(&a.oauth2DefaultScopes, "oauth2.default-scope", nil, "OAuth2 default scope in name=description format (repeatable)")

	// hmac
	flags.StringVar(&a.hmacName, "hmac.name", "", "Header/query/cookie name for HMAC signature (required when type=hmac)")
	flags.StringVar(&a.hmacIn, "hmac.in", "", "Where to send the HMAC signature: header, query or cookie (required when type=hmac)")
	flags.StringVar(&a.hmacAlgorithm, "hmac.algorithm", "", "HMAC algorithm: sha1, sha256, sha512 (required when type=hmac)")
	flags.StringVar(&a.hmacEncoding, "hmac.encoding", "hex", "HMAC encoding: hex, base64, base64url")
	flags.StringArrayVar(&a.hmacDigestInputs, "hmac.digest-input", nil, "HMAC digest input: body, url (repeatable)")

	_ = cmd.MarkFlagRequired("system-id")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func (a *addAuthScheme) Run(cmd *cobra.Command, _ []string) {
	a.configFactory.LoadConfigAndContext()

	payload := a.buildPayload(a.name)

	resp := v1.AuthSchemeConfig{}
	reqErr := a.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		WithPath("o/:organisation/systems/" + a.systemID + "/auth-scheme-configs/" + a.name).
		Into(&resp).
		JSONBody(payload).
		Do()
	if reqErr != nil {
		utils.NewExitError().WithMessage("failed to upsert auth scheme config").WithReason(reqErr).Done()
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Auth scheme config added/updated successfully")
}

func (a *addAuthScheme) buildPayload(id string) v1.UpsertAuthSchemeConfigJSONRequestBody {
	schemeType := v1.AuthSchemeType(a.schemeType)
	payload := v1.AuthSchemeConfig{Type: schemeType}

	switch schemeType {
	case v1.AuthSchemeTypeNone:
		payload.None = &v1.AuthSchemeConfigNone{ID: id, Description: a.description}

	case v1.AuthSchemeTypeBasicAuth:
		payload.BasicAuth = &v1.AuthSchemeConfigBasicAuth{ID: id, Description: a.description}

	case v1.AuthSchemeTypeCertificate:
		payload.Certificate = &v1.AuthSchemeConfigCertificate{ID: id, Description: a.description}

	case v1.AuthSchemeTypeApiKey:
		if a.apiKeyName == "" {
			utils.NewExitError().WithMessage("--api-key.name is required when type=api-key").Done()
		}
		if a.apiKeyIn == "" {
			utils.NewExitError().WithMessage("--api-key.in is required when type=api-key (header, query, cookie)").Done()
		}
		payload.ApiKey = &v1.AuthSchemeConfigAPIKey{
			ID:          id,
			Description: a.description,
			Name:        a.apiKeyName,
			In:          v1.AuthSchemeConfigAPIKeyIn(a.apiKeyIn),
		}

	case v1.AuthSchemeTypeOauth2:
		payload.Oauth2 = a.buildOAuth2Config(id)

	case v1.AuthSchemeTypeHmac:
		payload.Hmac = a.buildHMACConfig(id)
	}

	return payload
}

func (a *addAuthScheme) buildOAuth2Config(id string) *v1.AuthSchemeConfigOAuth2 {
	if a.oauth2TokenURL == "" {
		utils.NewExitError().WithMessage("--oauth2.token-url is required when type=oauth2").Done()
	}

	grant := a.buildOAuth2Grant()
	scopes := buildOAuth2Scopes(a.oauth2Scopes)

	cfg := &v1.AuthSchemeConfigOAuth2{
		ID:            id,
		Description:   a.description,
		TokenURL:      a.oauth2TokenURL,
		Grant:         grant,
		Scopes:        scopes,
		DefaultScopes: a.oauth2DefaultScopes,
	}
	if a.oauth2AuthorizeURL != "" {
		cfg.AuthorizeURL = a.oauth2AuthorizeURL
	}
	return cfg
}

func (a *addAuthScheme) buildOAuth2Grant() v1.AuthSchemeConfigOAuth2Grant {
	grantType := v1.AuthSchemeConfigOAuth2GrantType(a.oauth2GrantType)
	grant := v1.AuthSchemeConfigOAuth2Grant{Type: grantType}

	switch grantType {
	case v1.AuthSchemeConfigOAuth2GrantTypeAuthorizationCode:
		ac := &v1.AuthSchemeConfigOAuth2GrantAuthorizationCode{}
		if a.oauth2ClientID != "" {
			ac.ClientID = &a.oauth2ClientID
		}
		if a.oauth2ClientSecret != "" {
			ac.ClientSecret = &a.oauth2ClientSecret
		}
		grant.AuthorizationCode = ac
	case v1.AuthSchemeConfigOAuth2GrantTypePassword:
		pw := &v1.AuthSchemeConfigOAuth2GrantPassword{}
		if a.oauth2ClientID != "" {
			pw.ClientID = &a.oauth2ClientID
		}
		if a.oauth2ClientSecret != "" {
			pw.ClientSecret = &a.oauth2ClientSecret
		}
		grant.Password = pw
	case v1.AuthSchemeConfigOAuth2GrantTypeClientCredentials:
		cc := v1.AuthSchemeConfigOAuth2GrantClientCredentials{}
		grant.ClientCredentials = &cc
	}

	return grant
}

func buildOAuth2Scopes(rawScopes []string) []v1.OAuth2Scope {
	scopes := make([]v1.OAuth2Scope, 0, len(rawScopes))
	for _, s := range rawScopes {
		parts := strings.SplitN(s, "=", 2)
		scope := v1.OAuth2Scope{Name: parts[0]}
		if len(parts) == 2 {
			scope.Description = parts[1]
		}
		scopes = append(scopes, scope)
	}
	return scopes
}

func (a *addAuthScheme) buildHMACConfig(id string) *v1.AuthSchemeConfigHMAC {
	if a.hmacName == "" {
		utils.NewExitError().WithMessage("--hmac.name is required when type=hmac").Done()
	}
	if a.hmacIn == "" {
		utils.NewExitError().WithMessage("--hmac.in is required when type=hmac (header, query, cookie)").Done()
	}
	if a.hmacAlgorithm == "" {
		utils.NewExitError().WithMessage("--hmac.algorithm is required when type=hmac (sha1, sha256, sha512)").Done()
	}

	digestInputs := make([]v1.AuthSchemeConfigHMACDigestInputs, 0, len(a.hmacDigestInputs))
	for _, di := range a.hmacDigestInputs {
		digestInputs = append(digestInputs, v1.AuthSchemeConfigHMACDigestInputs(di))
	}
	if len(digestInputs) == 0 {
		digestInputs = []v1.AuthSchemeConfigHMACDigestInputs{v1.AuthSchemeConfigHMACDigestInputsBody}
	}

	return &v1.AuthSchemeConfigHMAC{
		ID:           id,
		Description:  a.description,
		Name:         a.hmacName,
		In:           v1.AuthSchemeConfigHMACIn(a.hmacIn),
		Algorithm:    v1.AuthSchemeConfigHMACAlgorithm(a.hmacAlgorithm),
		Encoding:     v1.AuthSchemeConfigHMACEncoding(a.hmacEncoding),
		DigestInputs: digestInputs,
	}
}
