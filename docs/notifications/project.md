---
title: "notifications project"
description: "Link, unlink, and list notification channels on a project + environment"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `link` | Link an existing notification channel to this project + environment |
| `list` | List notifications bound to a project (optionally filtered by environment) |
| `unlink` | Unlink a notification channel from this project + environment |

---

### `versori notifications project link`


Link an existing notification channel (created with 'versori notifications channels create')
to a project + environment. After linking, issues created in that environment by workflow code
('ctx.createIssue()' or '.catch()' blocks) also trigger an email through the channel. Issues
are always visible in the Issues UI without a linked channel — linking only adds email delivery.

If --channel-id or --environment is omitted, the CLI presents an interactive picker of the
available channels/environments by name. --project defaults from .versori when inside a synced
project directory.

```sh
versori notifications project link --channel-id <id> --environment <name> [--name <label>] [--project <project-id>] [flags]
```


**Flags:**
* `--channel-id`: ULID of the notification channel to link (prompts a picker if omitted)

* `--environment`: Name of the project environment (e.g. production, staging; prompts a picker if omitted)

* `-h`, `--help`: help for link
* `--name`: Display name for this link (defaults to the channel name)
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori notifications project list`




```sh
versori notifications project list [--project <project-id>] [--environment <name>] [flags]
```


**Flags:**
* `--environment`: Filter by environment name (e.g. production, staging)
* `-h`, `--help`: help for list
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori notifications project unlink`


Remove a project-notification binding so issues raised in the environment no longer alert
through the linked channel. The channel itself is not deleted; remove it separately with
'versori notifications channels delete'.

If --notification-id is omitted, the CLI lists existing bindings for the environment and prompts
you to pick one. If --environment is omitted, the CLI prompts you to pick an environment.
Confirms before deleting unless --yes is passed.

```sh
versori notifications project unlink --notification-id <id> --environment <name> [--project <project-id>] [--yes] [flags]
```


**Flags:**
* `--environment`: Name of the project environment the binding belongs to (e.g. production, staging)

* `-h`, `--help`: help for unlink
* `--notification-id`: ULID of the project-notification binding to remove
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `-y`, `--yes`: Skip the confirmation prompt



---
