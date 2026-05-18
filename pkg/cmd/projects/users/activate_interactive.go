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
	"sort"
	"strconv"
	"strings"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/ulid"
	"github.com/versori/cli/pkg/utils"
)

// projectConnectionTemplate is the slimmed-down view of one row from
// GET /o/:org/projects/:projectId/connection-templates?env_id=:envId — locally redeclared rather
// than imported from the `systems` command package to keep the dependency graph linear (users → API
// types) without dragging another sibling command package into the build.
type projectConnectionTemplate struct {
	Id                   string `json:"id"`                   // org-level system ID
	ConnectionTemplateId string `json:"connectionTemplateId"` // template ID used as the LHS of --connection
	Name                 string `json:"name"`                 // human name (e.g. "Mirakl")
	Dynamic              bool   `json:"dynamic"`              // true → embedded per-end-user connection expected
}

type projectConnectionTemplatesResponse struct {
	Items []projectConnectionTemplate `json:"items"`
}

// promptForConnections walks each connection template on the environment and asks the user to
// pick one of their existing connections per template. Used when --connection is omitted entirely
// so the user can drive the activation by selection rather than by ULID copy-paste.
//
// Auto-selects when only one candidate exists for a template (with a single-line "using …"
// notice) so the picker isn't a one-option click-through. Fails fast — with a copy-pasteable
// `versori connections create ...` hint — when a template has no matching connections, because
// activation can't proceed until the user creates one.
func (a *activate) promptForConnections(projectId, envId string) ([]connectionPair, error) {
	templates := a.fetchProjectConnectionTemplates(projectId, envId)
	if len(templates) == 0 {
		return nil, fmt.Errorf(
			"environment has no connection templates declared. Link a system to this environment first: " +
				"versori projects systems link --project " + projectId + " --environment " + a.environmentName + " --system <system-id>")
	}

	userULID, err := a.resolveUserULID()
	if err != nil {
		return nil, err
	}

	out := make([]connectionPair, 0, len(templates))
	for _, tpl := range templates {
		conns := a.fetchConnectionsForTemplate(tpl, userULID)
		if len(conns) == 0 {
			return nil, fmt.Errorf(
				"no %s connections exist for system %q.\n"+
					"  Create one first:  versori connections create --system %s%s --name <name> --field <key>=<value>...",
				connectionKindLabel(tpl.Dynamic), tpl.Name, tpl.Id, externalIdFragment(tpl.Dynamic, a.userExternalId))
		}

		connID, err := pickConnection(tpl, conns)
		if err != nil {
			return nil, err
		}

		tplID, err := ulid.Parse(tpl.ConnectionTemplateId)
		if err != nil {
			return nil, fmt.Errorf("invalid connection template ID %q on system %q: %w", tpl.ConnectionTemplateId, tpl.Name, err)
		}

		out = append(out, connectionPair{
			EnvironmentSystemID:  tplID,
			ExistingConnectionID: &connID,
		})
	}

	return out, nil
}

// pickConnection renders the picker for one template. Returns the chosen connection's ULID.
// Single-candidate templates auto-select (printed to stdout) so the user isn't forced through
// a one-option click-through; multi-candidate templates show the picker keyed by a label that
// includes the connection name and ULID so two same-named connections can still be distinguished.
func pickConnection(tpl projectConnectionTemplate, conns []v1.Connection) (ulid.ULID, error) {
	if len(conns) == 1 {
		fmt.Printf("Using connection %q (%s) for system %q (only candidate)\n", conns[0].Name, conns[0].ID.String(), tpl.Name)
		return conns[0].ID, nil
	}

	sel := elements.NewListSelect(fmt.Sprintf("Pick a connection for system %q (template %s):", tpl.Name, tpl.ConnectionTemplateId))
	for _, c := range conns {
		label := fmt.Sprintf("%s  (%s)", c.Name, c.ID.String())
		if c.BaseURL != "" {
			label = fmt.Sprintf("%s  (%s, %s)", c.Name, c.ID.String(), c.BaseURL)
		}
		sel.AddOption(label, c.ID.String())
	}

	var picked string
	if err := sel.Select(&picked); err != nil {
		return ulid.ULID{}, fmt.Errorf("connection picker failed: %w", err)
	}

	id, err := ulid.Parse(picked)
	if err != nil {
		return ulid.ULID{}, fmt.Errorf("invalid selected connection ID %q: %w", picked, err)
	}
	return id, nil
}

// promptForMissingRequiredVariables tops up `supplied` with values for any required schema keys
// the user didn't pass via --variable / --variables-file. Prompts are type-aware so the input
// matches the schema:
//   - boolean → Y/N confirm
//   - integer / number → validated single-line editor
//   - object / array  → multi-line editor expecting JSON
//   - string + unknown → plain single-line editor
//
// Optional schema keys are intentionally not prompted; only `required` entries surface, to keep
// the interactive flow as short as the activation actually demands.
func (a *activate) promptForMissingRequiredVariables(projectId string, supplied v1.DynamicVariables) (v1.DynamicVariables, error) {
	raw := json.RawMessage{}
	err := a.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&raw).
		WithPath("o/:organisation/projects/" + projectId + "/variables").
		Do()
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return supplied, nil
	}

	var schema struct {
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return supplied, nil
	}

	var missing []string
	for _, key := range schema.Required {
		if _, ok := supplied[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) == 0 {
		return supplied, nil
	}
	sort.Strings(missing)

	// Don't mutate caller's map — produce a fresh copy with prompted values merged in.
	out := make(v1.DynamicVariables, len(supplied)+len(missing))
	for k, v := range supplied {
		out[k] = v
	}

	for _, key := range missing {
		prop := schema.Properties[key]
		value, err := promptForVariable(key, prop.Type, prop.Description)
		if err != nil {
			return nil, fmt.Errorf("prompt for required variable %q failed: %w", key, err)
		}
		out[key] = value
	}

	return out, nil
}

// promptForVariable returns a value matching the schema's declared type for one variable, asking
// the user via the most-appropriate huh primitive. Numeric / JSON prompts validate on the input
// string so the user can't submit unparseable values and have the activation fail later.
func promptForVariable(key, typ, description string) (any, error) {
	title := key
	if description != "" {
		title = fmt.Sprintf("%s — %s", key, description)
	}
	if typ != "" {
		title = fmt.Sprintf("%s (required, type=%s)", title, typ)
	} else {
		title = fmt.Sprintf("%s (required)", title)
	}

	switch typ {
	case "boolean":
		var v bool
		if err := elements.NewConfirm(title).Confirm(&v); err != nil {
			return nil, err
		}
		return v, nil

	case "integer":
		var raw string
		editor := elements.NewEditor(title, false, elements.WithValidation(func(s string) error {
			if _, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err != nil {
				return fmt.Errorf("must be an integer")
			}
			return nil
		}))
		if err := editor.Edit(&raw); err != nil {
			return nil, err
		}
		n, _ := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		return n, nil

	case "number":
		var raw string
		editor := elements.NewEditor(title, false, elements.WithValidation(func(s string) error {
			if _, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err != nil {
				return fmt.Errorf("must be a number")
			}
			return nil
		}))
		if err := editor.Edit(&raw); err != nil {
			return nil, err
		}
		f, _ := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		return f, nil

	case "object", "array":
		var raw string
		editor := elements.NewEditor(title+" (enter JSON)", true, elements.WithValidation(func(s string) error {
			var v any
			if err := json.Unmarshal([]byte(s), &v); err != nil {
				return fmt.Errorf("must be valid JSON: %w", err)
			}
			return nil
		}))
		if err := editor.Edit(&raw); err != nil {
			return nil, err
		}
		var v any
		_ = json.Unmarshal([]byte(raw), &v)
		return v, nil

	default:
		var v string
		if err := elements.NewEditor(title, false).Edit(&v); err != nil {
			return nil, err
		}
		return v, nil
	}
}

// fetchProjectConnectionTemplates lists the environment's connection templates (the LHS of each
// --connection pair). Exits on API failure since this is core data; without it the picker has
// nothing to iterate.
func (a *activate) fetchProjectConnectionTemplates(projectId, envId string) []projectConnectionTemplate {
	resp := projectConnectionTemplatesResponse{}
	err := a.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/projects/"+projectId+"/connection-templates").
		WithQueryParam("env_id", envId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list project connection templates").WithReason(err).Done()
	}
	return resp.Items
}

// fetchConnectionsForTemplate returns the candidate connections for one template. Dynamic
// templates expect a per-end-user embedded connection so we filter by both system and end-user;
// non-dynamic templates expect a shared static connection so we filter by system only — matching
// how `versori connections list` would surface each kind.
func (a *activate) fetchConnectionsForTemplate(tpl projectConnectionTemplate, userULID string) []v1.Connection {
	resp := v1.ConnectionPage{}
	req := a.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/connections").
		WithQueryParam("system_id", tpl.Id)
	if tpl.Dynamic {
		req = req.WithQueryParam("end_user_id", userULID)
	}
	if err := req.Do(); err != nil {
		utils.NewExitError().
			WithMessage("failed to list connections for system " + tpl.Name).
			WithReason(err).
			Done()
	}
	return resp.Items
}

// resolveUserULID converts the user's external ID (a.userExternalId) into the platform ULID, the
// form the connections endpoint expects when filtering. Returns an error wrapped with the
// external ID so the user immediately sees which user couldn't be resolved.
func (a *activate) resolveUserULID() (string, error) {
	resp := v1.EndUserPage{}
	err := a.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/users").
		Do()
	if err != nil {
		return "", fmt.Errorf("failed to list users while resolving %q: %w", a.userExternalId, err)
	}
	for _, u := range resp.Users {
		if u.ExternalID == a.userExternalId {
			return u.ID.String(), nil
		}
	}
	return "", fmt.Errorf("no end-user found with external ID %q. Create one with 'versori users create -e %s -n <display-name>' first",
		a.userExternalId, a.userExternalId)
}

func connectionKindLabel(dynamic bool) string {
	if dynamic {
		return "embedded (per-end-user)"
	}
	return "static (project-wide)"
}

func externalIdFragment(dynamic bool, externalId string) string {
	if dynamic {
		return " --external-id " + externalId
	}
	return ""
}
