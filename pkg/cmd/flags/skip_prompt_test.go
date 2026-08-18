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

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestAddSkipPromptFlagsAcceptsYesAndConfirm(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "neither", args: nil, want: false},
		{name: "yes", args: []string{"--yes"}, want: true},
		{name: "short y", args: []string{"-y"}, want: true},
		{name: "confirm", args: []string{"--confirm"}, want: true},
		{name: "both", args: []string{"--yes", "--confirm"}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var skip bool
			f := pflag.NewFlagSet("skip-prompt", pflag.ContinueOnError)
			AddSkipPromptFlags(f, &skip)
			if err := f.Parse(tc.args); err != nil {
				t.Fatalf("parse %v: %v", tc.args, err)
			}
			if skip != tc.want {
				t.Fatalf("skip=%v, want %v for %v", skip, tc.want, tc.args)
			}
		})
	}
}
