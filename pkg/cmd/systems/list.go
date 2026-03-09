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
	"net/http"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type list struct {
	configFactory *config.ConfigFactory
}

// light-weight printable type registration for table rendering
// We register the reduced view to avoid printer panic and control columns.
type printableSystem struct {
	Name              string                `json:"name"`
	Id                string                `json:"id"`
	Domain            string                `json:"domain"`
	TemplateBaseUrl   string                `json:"templateBaseUrl"`
	AuthSchemeConfigs []v1.AuthSchemeConfig `json:"authSchemeConfigs" yaml:"authSchemeConfigs"`
}

func init() {
	// Only print the items fields we care about by default in tables
	utils.RegisterResource(printableSystem{}, []string{"Name", "Id", "Domain", "TemplateBaseUrl"})
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists systems for the current organisation context",
		Run:   l.Run,
	}

	return cmd
}

func (l *list) Run(cmd *cobra.Command, args []string) {
	resp := v1.SystemPage{}
	err := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/systems").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list systems").WithReason(err).Done()
	}

	items := make([]printableSystem, 0, len(resp.Items))
	for _, it := range resp.Items {
		items = append(items, printableSystem{
			Name:              it.Name,
			Id:                it.ID.String(),
			Domain:            it.Domain,
			TemplateBaseUrl:   it.TemplateBaseUrl,
			AuthSchemeConfigs: it.AuthSchemeConfigs,
		})
	}

	l.configFactory.Print(items)
}
