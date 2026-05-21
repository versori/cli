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

package channels

import (
	"net/http"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type printableChannel struct {
	Name string `json:"name"`
	Id   string `json:"id"`
	Type string `json:"type"`
	To   string `json:"to,omitempty"`
}

func init() {
	utils.RegisterResource(printableChannel{}, []string{"Name", "Id", "Type", "To"})
}

type list struct {
	configFactory *config.ConfigFactory
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{configFactory: c}

	return &cobra.Command{
		Use:   "list",
		Short: "List notification channels in the current organisation",
		Run:   l.Run,
	}
}

func (l *list) Run(cmd *cobra.Command, args []string) {
	resp := v1.NotificationChannelList{}
	err := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/notification_channels").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list notification channels").WithReason(err).Done()
	}

	items := make([]printableChannel, 0, len(resp.Items))
	for _, ch := range resp.Items {
		to := ""
		if ch.Config.Email != nil {
			to = ch.Config.Email.To
		}

		items = append(items, printableChannel{
			Name: ch.Name,
			Id:   ch.Id.String(),
			Type: ch.Type,
			To:   to,
		})
	}

	l.configFactory.Print(items)
}
