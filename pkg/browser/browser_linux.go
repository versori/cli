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

package browser

import (
	"fmt"
	"os/exec"
)

// OpenURL opens the given URL in the default browser.
func OpenURL(url string) error {
	cmdsToTry := []string{"xdg-open", "x-www-browser"}

	for i, _ := range cmdsToTry {
		cmd, err := exec.LookPath(cmdsToTry[i])
		if err != nil {
			continue
		}

		return exec.Command(cmd, url).Start()
	}

	return fmt.Errorf("You need to install one of the following to open a browser: %v", cmdsToTry)
}
