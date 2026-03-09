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
	"github.com/versori/cli/pkg/utils"
)

type list struct {
	configFactory *config.ConfigFactory

	// filters
	systemID  string
	endUserID string
}

// light-weight printable type registration for table rendering
// We register the reduced view to avoid printer panic and control columns.
type printableItem struct {
	Name     string `json:"name"`
	Id       string `json:"id"`
	SystemId string `json:"systemId"`
	BaseUrl  string `json:"baseUrl"`
}

func init() {
	// Only print the items fields we care about by default in tables
	utils.RegisterResource(printableItem{}, []string{"Name", "Id", "SystemId", "BaseUrl"})
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists connections for the current organisation context",
		Run:   l.Run,
	}

	cmd.Flags().StringVar(&l.systemID, "system", "", "Filter by system ID")
	cmd.Flags().StringVar(&l.endUserID, "end-user", "", "Filter by end user ID")

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
		req = req.WithQueryParam("end_user_id", l.endUserID)
	}

	if err := req.Do(); err != nil {
		utils.NewExitError().WithMessage("failed to list connections").WithReason(err).Done()
	}

	items := make([]printableItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		items = append(items, printableItem{
			Name:     it.Name,
			Id:       it.ID.String(),
			SystemId: it.SystemID.String(),
			BaseUrl:  it.BaseURL,
		})
	}

	l.configFactory.Print(items)
}
