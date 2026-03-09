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

package executionpools

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

type printableExecutionPool struct {
	Name string `json:"name"`
}

func init() {
	utils.RegisterResource(printableExecutionPool{}, []string{"Name"})
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists available execution pools for the current organisation",
		Run:   l.Run,
	}

	return cmd
}

func (l *list) Run(cmd *cobra.Command, args []string) {
	resp := v1.ExecutionPoolList{}
	err := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/execution-pools").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list execution pools").WithReason(err).Done()
	}

	items := make([]printableExecutionPool, 0, len(resp.ExecutionPools))
	for _, ep := range resp.ExecutionPools {
		items = append(items, printableExecutionPool{
			Name: ep.Name,
		})
	}

	l.configFactory.Print(items)
}
