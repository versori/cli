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
	"net/http"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type list struct {
	configFactory *config.ConfigFactory
}

type printableUser struct {
	DisplayName string `json:"displayName"`
	ExternalId  string `json:"externalId"`
	Id          string `json:"id"`
	CreatedAt   string `json:"createdAt"`
}

func init() {
	utils.RegisterResource(printableUser{}, []string{"DisplayName", "ExternalId", "Id", "CreatedAt"})
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists users in the current organisation context",
		Run:   l.Run,
	}

	return cmd
}

func (l *list) Run(cmd *cobra.Command, args []string) {
	resp := v1.EndUserPage{}
	err := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/users").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list users").WithReason(err).Done()
	}

	items := make([]printableUser, 0, len(resp.Users))
	for _, u := range resp.Users {
		items = append(items, printableUser{
			DisplayName: u.DisplayName,
			ExternalId:  u.ExternalID,
			Id:          u.ID.String(),
			CreatedAt:   u.CreatedAt.String(),
		})
	}

	l.configFactory.Print(items)
}
