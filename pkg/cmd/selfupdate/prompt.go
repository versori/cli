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
	"fmt"

	"github.com/versori/cli/pkg/cmd/elements"
)

const (
	promptVSCode = "Install the Versori extension into VS Code?"
	promptCursor = "Install the Versori extension into Cursor?"
	promptBoth   = "Install the Versori extension into:"

	selectBoth   = "Both"
	selectCancel = "Cancel"
)

var (
	errNoEditors = errors.New("neither VS Code nor Cursor CLI was found; pass --vscode-path or --cursor-path")
	errNeedTTY   = errors.New("a TTY or --yes is required to install the extension")
	errCancelled = errors.New("cancelled; no changes were made")
)

func chooseEditors(resolved []Editor, yes bool, stdinTTY bool) ([]Editor, error) {
	if len(resolved) == 0 {
		return nil, errNoEditors
	}
	if yes {
		return resolved, nil
	}
	if !stdinTTY {
		return nil, errNeedTTY
	}

	if len(resolved) == 1 {
		return confirmOneEditor(resolved[0])
	}

	sel := elements.NewListSelect(promptBoth)
	sel.AddOption(string(EditorVSCode), string(EditorVSCode))
	sel.AddOption(string(EditorCursor), string(EditorCursor))
	sel.AddOption(selectBoth, selectBoth)
	sel.AddOption(selectCancel, selectCancel)

	var picked string
	if err := sel.Select(&picked); err != nil {
		return nil, fmt.Errorf("failed to read selection: %w", err)
	}

	switch picked {
	case selectCancel, "":
		return nil, errCancelled
	case selectBoth:
		return resolved, nil
	case string(EditorVSCode), string(EditorCursor):
		return filterEditors(resolved, EditorKind(picked)), nil
	default:
		return nil, errCancelled
	}
}

func confirmOneEditor(ed Editor) ([]Editor, error) {
	prompt := promptVSCode
	if ed.Kind == EditorCursor {
		prompt = promptCursor
	}

	confirmed := false
	if err := elements.NewConfirm(prompt).Confirm(&confirmed); err != nil {
		return nil, fmt.Errorf("failed to read confirmation: %w", err)
	}
	if !confirmed {
		return nil, nil
	}
	return []Editor{ed}, nil
}

func filterEditors(editors []Editor, kind EditorKind) []Editor {
	var out []Editor
	for _, ed := range editors {
		if ed.Kind == kind {
			out = append(out, ed)
		}
	}
	return out
}
