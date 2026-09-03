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
	"errors"
	"strings"
	"testing"
)

func TestInstallVSIXPassesForce(t *testing.T) {
	var gotName string
	var gotArgs []string
	err := InstallVSIX("/path/to/code", "/tmp/ext.vsix", func(name string, args ...string) ([]byte, error) {
		gotName = name
		gotArgs = append([]string{}, args...)
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotName != "/path/to/code" {
		t.Fatalf("bin=%q", gotName)
	}
	want := []string{"--install-extension", "/tmp/ext.vsix", "--force"}
	if strings.Join(gotArgs, " ") != strings.Join(want, " ") {
		t.Fatalf("args=%v; want %v", gotArgs, want)
	}
}

func TestInstallVSIXRunError(t *testing.T) {
	err := InstallVSIX("/path/to/code", "/tmp/ext.vsix", func(name string, args ...string) ([]byte, error) {
		return []byte("denied"), errors.New("exit 1")
	})
	if err == nil {
		t.Fatal("want error")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Fatalf("err=%v", err)
	}
}

func TestHasExtension(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		ok, err := HasExtension("/path/to/code", ExtensionID, func(name string, args ...string) ([]byte, error) {
			if len(args) != 1 || args[0] != "--list-extensions" {
				t.Fatalf("args=%v", args)
			}
			return []byte("foo.bar\nversori.versori-vscode\nbaz.qux\n"), nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("HasExtension: want true")
		}
	})

	t.Run("false", func(t *testing.T) {
		ok, err := HasExtension("/path/to/code", ExtensionID, func(name string, args ...string) ([]byte, error) {
			return []byte("foo.bar\nbaz.qux\n"), nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatal("HasExtension: want false")
		}
	})
}
