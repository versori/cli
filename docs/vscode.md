---
title: "vscode"
description: "Manage the Versori VS Code and Cursor extension"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `install` | Install the Versori extension into VS Code or Cursor |

---

### `versori vscode install`


Install the Versori VS Code / Cursor extension.

Downloads the latest vsix compatible with this CLI and installs it with
the editor CLI (--install-extension --force).

With one resolved editor, installs immediately. If both VS Code and Cursor
are resolved, prompts you to choose where to install. Pass --yes to install
into every resolved editor; non-interactive shells with both editors require
--yes or --confirm.

Use --vscode-path and --cursor-path to point at editor CLIs that are not
on PATH.

```sh
versori vscode install [flags]
```


**Flags:**
* `--confirm`: Skip the confirmation prompt (same as --yes)
* `--cursor-path`: Absolute path to the Cursor CLI (cursor)
* `-h`, `--help`: help for install
* `--vscode-path`: Absolute path to the VS Code CLI (code)
* `-y`, `--yes`: Skip the confirmation prompt



---
