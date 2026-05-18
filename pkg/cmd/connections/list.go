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
	"net/http"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/ulid"
	"github.com/versori/cli/pkg/utils"
)

type list struct {
	configFactory *config.ConfigFactory

	// filters
	systemID  string
	endUserID string
}

// printableItem is the reduced view rendered as a table row. We split the connection's own
// identity (ConnectionName / ConnectionId) from the underlying system's (SystemName / SystemId) so
// the table doesn't bury what's-which under generic Name / Id columns. SystemName is resolved at
// list-time from a single GET /o/:org/systems lookup and is best-effort — empty if the lookup
// fails so SystemId stays visible regardless.
type printableItem struct {
	ConnectionName string `json:"connectionName"`
	ConnectionId   string `json:"connectionId"`
	SystemName     string `json:"systemName,omitempty"`
	SystemId       string `json:"systemId"`
	BaseUrl        string `json:"baseUrl,omitempty"`
}

func init() {
	utils.RegisterResource(printableItem{}, []string{"ConnectionName", "ConnectionId", "SystemName", "SystemId", "BaseUrl"})
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists connections for the current organisation context",
		Run:   l.Run,
	}

	cmd.Flags().StringVar(&l.systemID, "system", "", "Filter by system ID")
	cmd.Flags().StringVar(&l.endUserID, "end-user", "", "Filter by end-user ULID or external ID (external IDs are resolved client-side)")

	return cmd
}

func (l *list) Run(cmd *cobra.Command, args []string) {
	resp := v1.ConnectionPage{}
	req := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/connections")

	if l.systemID != "" {
		req = req.WithQueryParam("system_id", l.systemID)
	}
	if l.endUserID != "" {
		req = req.WithQueryParam("end_user_id", l.resolveEndUserID(l.endUserID))
	}

	if err := req.Do(); err != nil {
		utils.NewExitError().WithMessage("failed to list connections").WithReason(err).Done()
	}

	systemNameById := l.fetchSystemNames(resp.Items)

	items := make([]printableItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		items = append(items, printableItem{
			ConnectionName: it.Name,
			ConnectionId:   it.ID.String(),
			SystemName:     systemNameById[it.SystemID.String()],
			SystemId:       it.SystemID.String(),
			BaseUrl:        it.BaseURL,
		})
	}

	l.configFactory.Print(items)
}

// resolveEndUserID takes whatever the user passed to --end-user and returns a ULID suitable for
// the platform's end_user_id query param. If the value parses as a ULID it's used verbatim;
// otherwise the org's end-users are listed and the entry whose ExternalID matches is resolved
// to its ULID. This mirrors how `versori projects users details / deactivate / activate` already
// accept an external ID — the platform itself only accepts a ULID here, but the friendlier
// `--end-user mirakl` form is what users intuit (and what the skill docs describe).
//
// Fails loudly with a message naming the unresolved value so the user immediately sees that
// neither a ULID parse nor an external-ID match worked, rather than the platform's opaque
// "not a valid ULID" rejection.
func (l *list) resolveEndUserID(value string) string {
	if _, err := ulid.Parse(value); err == nil {
		return value
	}

	resp := v1.EndUserPage{}
	err := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/users").
		Do()
	if err != nil {
		utils.NewExitError().
			WithMessage("failed to resolve --end-user " + value + " (not a ULID; tried external-ID lookup)").
			WithReason(err).
			Done()
	}

	for _, u := range resp.Users {
		if u.ExternalID == value {
			return u.ID.String()
		}
	}

	utils.NewExitError().
		WithMessage("no end-user found with ULID or external ID matching " + value).
		Done()

	return "" // unreachable
}

// fetchSystemNames returns a systemId → systemName lookup so the table can render the friendly
// name beside the ULID. Best-effort: a failed / empty systems list collapses to nil and the
// caller falls back to showing SystemId only — never aborts the connections list itself, since
// the connection data the user actually asked for is already in hand.
func (l *list) fetchSystemNames(conns []v1.Connection) map[string]string {
	if len(conns) == 0 {
		return nil
	}

	resp := v1.SystemPage{}
	err := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/systems").
		Do()
	if err != nil {
		return nil
	}

	out := make(map[string]string, len(resp.Items))
	for _, s := range resp.Items {
		out[s.ID.String()] = s.Name
	}
	return out
}
