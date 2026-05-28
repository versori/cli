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
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

const (
	emailChannelType = "email"
)

type create struct {
	configFactory *config.ConfigFactory
	name          string
	email         string
	cc            []string
}

func NewCreate(c *config.ConfigFactory) *cobra.Command {
	cr := &create{configFactory: c}

	cmd := &cobra.Command{
		Use:   "create --name <name> --email <addr> [--cc <addr>]...",
		Short: "Create an email notification channel for the current organisation",
		Long: `Create an email notification channel. Channels are organisation-scoped; bind one to a project
with 'versori notifications project link'.

Pass --email to set the primary recipient. Use --cc (repeatable) for additional recipients. The
CLI does not derive an email from the active context's token — service-key tokens carry no user
identity, so the recipient must be supplied explicitly.`,
		Run: cr.Run,
	}

	flags := cmd.Flags()
	flags.StringVar(&cr.name, "name", "", "Display name for the channel")
	flags.StringVar(&cr.email, "email", "", "Primary email address (required)")
	flags.StringSliceVar(&cr.cc, "cc", nil, "Additional addresses to CC (repeatable)")

	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func (c *create) Run(cmd *cobra.Command, _ []string) {
	if c.email == "" {
		utils.NewExitError().
			WithMessage("--email is required to create an email notification channel; pass --email <addr>").
			Done()
	}

	payload := v1.CreateNotificationChannelJSONRequestBody{
		Name: c.name,
		Type: emailChannelType,
		Config: v1.NotificationConfig{
			Email: &v1.EmailNotificationConfig{
				To: c.email,
				Cc: c.cc,
			},
		},
	}

	resp := v1.NotificationChannel{}
	err := c.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		Into(&resp).
		WithPath("o/:organisation/notification_channels").
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to create notification channel").WithReason(err).Done()
	}

	fmt.Printf("Created notification channel %q (id: %s, to: %s)\n", resp.Name, resp.Id.String(), c.email)
}
