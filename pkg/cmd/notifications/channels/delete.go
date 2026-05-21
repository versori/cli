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
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/utils"
)

type deleteChannel struct {
	configFactory *config.ConfigFactory
	channelId     string
	yes           bool
}

func NewDelete(c *config.ConfigFactory) *cobra.Command {
	d := &deleteChannel{configFactory: c}

	cmd := &cobra.Command{
		Use:     "delete --channel-id <id> [--yes]",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete an organisation-wide notification channel",
		Long: `Delete a notification channel from the current organisation. Any project bindings using this
channel will stop firing (delete the bindings first with 'versori notifications project unlink'
if you want a graceful tear-down).

If --channel-id is omitted, the CLI shows a picker of existing channels. Confirms before deleting
unless --yes is passed.`,
		Run: d.Run,
	}

	cmd.Flags().StringVar(&d.channelId, "channel-id", "", "ULID of the channel to delete (prompts a picker if omitted)")
	cmd.Flags().BoolVarP(&d.yes, "yes", "y", false, "Skip the confirmation prompt")

	return cmd
}

func (d *deleteChannel) Run(cmd *cobra.Command, _ []string) {
	channelName := d.resolveChannel()

	if !d.yes {
		confirmed := false
		err := elements.
			NewConfirm(fmt.Sprintf("Delete channel %q from this organisation?", channelName)).
			Confirm(&confirmed)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read confirmation").WithReason(err).Done()
		}

		if !confirmed {
			fmt.Println("Aborted; no changes were made.")

			return
		}
	}

	err := d.configFactory.
		NewRequest().
		WithMethod(http.MethodDelete).
		WithPath("o/:organisation/notification_channels/" + d.channelId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to delete notification channel").WithReason(err).Done()
	}

	fmt.Printf("Deleted channel %q.\n", channelName)
}

func (d *deleteChannel) resolveChannel() string {
	channels := d.fetchChannels()

	if d.channelId != "" {
		for _, ch := range channels {
			if ch.Id.String() == d.channelId {
				return ch.Name
			}
		}

		utils.NewExitError().WithMessage(fmt.Sprintf("channel %s not found in current organisation", d.channelId)).Done()
	}

	if len(channels) == 0 {
		utils.NewExitError().WithMessage("no notification channels exist").Done()
	}

	sel := elements.NewListSelect("Select a channel to delete:")
	for _, ch := range channels {
		to := ""
		if ch.Config.Email != nil {
			to = ch.Config.Email.To
		}

		label := fmt.Sprintf("%s  (%s)", ch.Name, to)
		sel.AddOption(label, ch.Id.String())
	}

	if err := sel.Select(&d.channelId); err != nil {
		utils.NewExitError().WithMessage("failed to read channel selection").WithReason(err).Done()
	}

	for _, ch := range channels {
		if ch.Id.String() == d.channelId {
			return ch.Name
		}
	}

	return d.channelId
}

func (d *deleteChannel) fetchChannels() []v1.NotificationChannel {
	resp := v1.NotificationChannelList{}
	err := d.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/notification_channels").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list notification channels").WithReason(err).Done()
	}

	return resp.Items
}
