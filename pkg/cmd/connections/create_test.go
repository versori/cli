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
	"testing"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/ulid"
)

func TestCreateCredentialData(t *testing.T) {
	orgID := ulid.MustDefault()
	configFactory := &config.ConfigFactory{
		Context: &config.Context{
			OrganisationId: orgID.String(),
		},
	}
	systemID := ulid.MustDefault().String()
	ctx := context.Background()

	tests := []struct {
		name             string
		fields           credentialFields
		authSchemeConfig v1.AuthSchemeConfig
		wantCredType     v1.CredentialType
		validateResult   func(t *testing.T, result v1.ConnectionCredential)
	}{
		{
			name: "api key authentication",
			fields: credentialFields{
				apiKey: "test-api-key-123",
			},
			authSchemeConfig: v1.AuthSchemeConfig{
				Type: v1.AuthSchemeTypeApiKey,
			},
			wantCredType: v1.CredentialTypeString,
			validateResult: func(t *testing.T, result v1.ConnectionCredential) {
				if result.Credential == nil {
					t.Fatal("credential is nil")
				}
				if result.Credential.Data == nil {
					t.Fatal("credential data is nil")
				}
				if result.Credential.Data.String == nil {
					t.Fatal("credential data string is nil")
				}
				if result.Credential.Data.String.Value != "test-api-key-123" {
					t.Errorf("got api key %q, want %q", result.Credential.Data.String.Value, "test-api-key-123")
				}
				if result.Credential.Type != v1.CredentialTypeString {
					t.Errorf("got credential type %v, want %v", result.Credential.Type, v1.CredentialTypeString)
				}
				if result.AuthSchemeConfig.Type != v1.AuthSchemeTypeApiKey {
					t.Errorf("got auth scheme type %v, want %v", result.AuthSchemeConfig.Type, v1.AuthSchemeTypeApiKey)
				}
			},
		},
		{
			name: "basic auth authentication",
			fields: credentialFields{
				username: "testuser",
				password: "testpass",
			},
			authSchemeConfig: v1.AuthSchemeConfig{
				Type: v1.AuthSchemeTypeBasicAuth,
			},
			wantCredType: v1.CredentialTypeBasicAuth,
			validateResult: func(t *testing.T, result v1.ConnectionCredential) {
				if result.Credential == nil {
					t.Fatal("credential is nil")
				}
				if result.Credential.Data == nil {
					t.Fatal("credential data is nil")
				}
				if result.Credential.Data.BasicAuth == nil {
					t.Fatal("credential data basic auth is nil")
				}
				if result.Credential.Data.BasicAuth.Username != "testuser" {
					t.Errorf("got username %q, want %q", result.Credential.Data.BasicAuth.Username, "testuser")
				}
				if result.Credential.Data.BasicAuth.Password != "testpass" {
					t.Errorf("got password %q, want %q", result.Credential.Data.BasicAuth.Password, "testpass")
				}
				if result.Credential.Type != v1.CredentialTypeBasicAuth {
					t.Errorf("got credential type %v, want %v", result.Credential.Type, v1.CredentialTypeBasicAuth)
				}
				if result.AuthSchemeConfig.Type != v1.AuthSchemeTypeBasicAuth {
					t.Errorf("got auth scheme type %v, want %v", result.AuthSchemeConfig.Type, v1.AuthSchemeTypeBasicAuth)
				}
			},
		},
		{
			name: "none authentication",
			fields: credentialFields{
				bypass: true,
			},
			authSchemeConfig: v1.AuthSchemeConfig{
				Type: v1.AuthSchemeTypeNone,
				None: &v1.AuthSchemeConfigNone{},
			},
			wantCredType: v1.CredentialTypeNone,
			validateResult: func(t *testing.T, result v1.ConnectionCredential) {
				if result.Credential == nil {
					t.Fatal("credential is nil")
				}
				if result.Credential.Data != nil {
					t.Errorf("credential data should be nil for none auth type, got %+v", result.Credential.Data)
				}
				if result.Credential.Type != v1.CredentialTypeNone {
					t.Errorf("got credential type %v, want %v", result.Credential.Type, v1.CredentialTypeNone)
				}
				if result.AuthSchemeConfig.Type != v1.AuthSchemeTypeNone {
					t.Errorf("got auth scheme type %v, want %v", result.AuthSchemeConfig.Type, v1.AuthSchemeTypeNone)
				}
				if result.AuthSchemeConfig.None == nil {
					t.Error("auth scheme config none is nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &create{
				configFactory: configFactory,
				fields:        tt.fields,
			}

			result := c.createCredentialData(ctx, systemID, tt.authSchemeConfig)

			if result.Credential.OrganisationID.String() != orgID.String() {
				t.Errorf("got organisation id %q, want %q", result.Credential.OrganisationID.String(), orgID.String())
			}

			tt.validateResult(t, result)
		})
	}
}

func TestHandleOAuth2(t *testing.T) {
	orgID := ulid.MustDefault()
	configFactory := &config.ConfigFactory{
		Context: &config.Context{
			OrganisationId: orgID.String(),
		},
	}
	systemID := ulid.MustDefault().String()
	ctx := context.Background()

	tests := []struct {
		name             string
		fields           credentialFields
		authSchemeConfig *v1.AuthSchemeConfigOAuth2
		wantCredType     v1.CredentialType
		validateResult   func(t *testing.T, credData v1.CredentialData, credType v1.CredentialType)
	}{
		{
			name: "oauth2 client credentials",
			fields: credentialFields{
				clientID:     "test-client-id",
				clientSecret: "test-client-secret",
				tokenUrl:     "",
			},
			authSchemeConfig: &v1.AuthSchemeConfigOAuth2{
				Grant: v1.AuthSchemeConfigOAuth2Grant{
					Type: v1.AuthSchemeConfigOAuth2GrantTypeClientCredentials,
				},
				TokenURL: "https://example.com/token",
			},
			wantCredType: v1.CredentialTypeOauth2Client,
			validateResult: func(t *testing.T, credData v1.CredentialData, credType v1.CredentialType) {
				if credData.Oauth2Client == nil {
					t.Fatal("oauth2 client credential data is nil")
				}
				if credData.Oauth2Client.ClientID != "test-client-id" {
					t.Errorf("got client id %q, want %q", credData.Oauth2Client.ClientID, "test-client-id")
				}
				if credData.Oauth2Client.ClientSecret != "test-client-secret" {
					t.Errorf("got client secret %q, want %q", credData.Oauth2Client.ClientSecret, "test-client-secret")
				}
				if credData.Oauth2Client.TokenURL != "https://example.com/token" {
					t.Errorf("got token url %q, want %q", credData.Oauth2Client.TokenURL, "https://example.com/token")
				}
				if credType != v1.CredentialTypeOauth2Client {
					t.Errorf("got credential type %v, want %v", credType, v1.CredentialTypeOauth2Client)
				}
			},
		},
		{
			name: "oauth2 client credentials with custom token url",
			fields: credentialFields{
				clientID:     "test-client-id",
				clientSecret: "test-client-secret",
				tokenUrl:     "https://custom.example.com/oauth/token",
			},
			authSchemeConfig: &v1.AuthSchemeConfigOAuth2{
				Grant: v1.AuthSchemeConfigOAuth2Grant{
					Type: v1.AuthSchemeConfigOAuth2GrantTypeClientCredentials,
				},
				TokenURL: "https://example.com/token",
			},
			wantCredType: v1.CredentialTypeOauth2Client,
			validateResult: func(t *testing.T, credData v1.CredentialData, credType v1.CredentialType) {
				if credData.Oauth2Client == nil {
					t.Fatal("oauth2 client credential data is nil")
				}
				if credData.Oauth2Client.TokenURL != "https://custom.example.com/oauth/token" {
					t.Errorf("got token url %q, want %q", credData.Oauth2Client.TokenURL, "https://custom.example.com/oauth/token")
				}
			},
		},
		{
			name: "oauth2 password grant",
			fields: credentialFields{
				username: "testuser",
				password: "testpass",
			},
			authSchemeConfig: &v1.AuthSchemeConfigOAuth2{
				Grant: v1.AuthSchemeConfigOAuth2Grant{
					Type: v1.AuthSchemeConfigOAuth2GrantTypePassword,
				},
				TokenURL: "https://example.com/token",
			},
			wantCredType: v1.CredentialTypeOauth2Password,
			validateResult: func(t *testing.T, credData v1.CredentialData, credType v1.CredentialType) {
				if credData.Oauth2Password == nil {
					t.Fatal("oauth2 password credential data is nil")
				}
				if credData.Oauth2Password.Username != "testuser" {
					t.Errorf("got username %q, want %q", credData.Oauth2Password.Username, "testuser")
				}
				if credData.Oauth2Password.Password != "testpass" {
					t.Errorf("got password %q, want %q", credData.Oauth2Password.Password, "testpass")
				}
				if credType != v1.CredentialTypeOauth2Password {
					t.Errorf("got credential type %v, want %v", credType, v1.CredentialTypeOauth2Password)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &create{
				configFactory: configFactory,
				fields:        tt.fields,
			}

			credData, credType := c.handleOAuth2(ctx, systemID, tt.authSchemeConfig)

			if credType != tt.wantCredType {
				t.Errorf("got credential type %v, want %v", credType, tt.wantCredType)
			}

			tt.validateResult(t, credData, credType)
		})
	}
}
