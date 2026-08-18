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

package flags

import "github.com/spf13/pflag"

const skipPromptUsage = "Skip the confirmation prompt"

// AddSkipPromptFlags registers --yes / -y and --confirm on dest. Either flag
// skips the TTY confirmation; they are aliases, not a dry-run gate.
func AddSkipPromptFlags(f *pflag.FlagSet, dest *bool) {
	f.BoolVarP(dest, "yes", "y", false, skipPromptUsage)
	f.BoolVar(dest, "confirm", false, skipPromptUsage+" (same as --yes)")
}
