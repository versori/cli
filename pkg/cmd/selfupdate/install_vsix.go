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
	"fmt"
	"os/exec"
	"strings"
)

const ExtensionID = "versori.versori-vscode"

type execRunner func(name string, args ...string) ([]byte, error)

func execRun(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func orExec(run execRunner) execRunner {
	if run != nil {
		return run
	}
	return execRun
}

func InstallVSIX(bin, vsixPath string, run execRunner) error {
	out, err := orExec(run)(bin, "--install-extension", vsixPath, "--force")
	if err != nil {
		if len(out) > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return err
	}
	return nil
}

func HasExtension(bin, id string, run execRunner) (bool, error) {
	out, err := orExec(run)(bin, "--list-extensions")
	if err != nil {
		if len(out) > 0 {
			return false, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return false, err
	}
	return strings.Contains(string(out), id), nil
}
