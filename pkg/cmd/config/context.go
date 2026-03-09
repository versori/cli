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

package config

type Context struct {
	Name           string `yaml:"name"`
	OrganisationId string `yaml:"organisation_id"`
	JWT            string `yaml:"jwt"`
	SigningKey     string `yaml:"signing_key"`
}
