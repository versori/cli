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

package selfupdate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/mod/semver"
)

const DefaultCompatURL = "https://raw.githubusercontent.com/versori/cli/main/vscode-compat.json"

const curlInstall = "curl -fsSL https://raw.githubusercontent.com/versori/cli/main/install.sh | sh"

type CompatTable struct {
	ExtensionID string          `json:"extension_id"`
	Releases    []CompatRelease `json:"releases"`
}

type CompatRelease struct {
	Vsix   string `json:"vsix"`
	MinCLI string `json:"min_cli"`
}

func ParseCompat(data []byte) (CompatTable, error) {
	var table CompatTable
	if err := json.Unmarshal(data, &table); err != nil {
		return CompatTable{}, err
	}
	return table, nil
}

func PickVSIX(cliVersion string, table CompatTable) (vsixTag string, ok bool) {
	cli := canonSemver(cliVersion)
	if !semver.IsValid(cli) {
		return "", false
	}

	var bestTag, bestCanon string
	for _, rel := range table.Releases {
		min := canonSemver(rel.MinCLI)
		vsix := canonSemver(rel.Vsix)
		if !semver.IsValid(min) || !semver.IsValid(vsix) {
			continue
		}
		if semver.Compare(min, cli) > 0 {
			continue
		}
		if bestCanon == "" || semver.Compare(vsix, bestCanon) > 0 {
			bestTag = rel.Vsix
			bestCanon = vsix
		}
	}
	if bestCanon == "" {
		return "", false
	}
	return bestTag, true
}

func TooOldMessage() string {
	return "this CLI is too old for any published extension; run versori update, or: " + curlInstall
}

func FetchCompat(client *http.Client, url string) (CompatTable, error) {
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(url)
	if err != nil {
		return CompatTable{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompatTable{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return CompatTable{}, fmt.Errorf("compat table: unexpected status %d", resp.StatusCode)
	}
	return ParseCompat(body)
}

func canonSemver(v string) string {
	v = strings.TrimSpace(v)
	if v != "" && v[0] != 'v' {
		return "v" + v
	}
	return v
}
