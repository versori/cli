---
title: "notifications channels"
description: "Manage organisation-wide notification channels"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `create` | Create an email notification channel for the current organisation |
| `delete` | Delete an organisation-wide notification channel |
| `list` | List notification channels in the current organisation |

---

### `versori notifications channels create`


Create an email notification channel. Channels are organisation-scoped; bind one to a project
with 'versori notifications project link'.

Pass --email to set the primary recipient. Use --cc (repeatable) for additional recipients. The
CLI does not derive an email from the active context's token — service-key tokens carry no user
identity, so the recipient must be supplied explicitly.

```sh
versori notifications channels create --name <name> --email <addr> [--cc <addr>]... [flags]
```


**Flags:**
* `--cc`: Additional addresses to CC (repeatable)
* `--email`: Primary email address (required)
* `-h`, `--help`: help for create
* `--name`: Display name for the channel



---

### `versori notifications channels delete`


Delete a notification channel from the current organisation. Any project bindings using this
channel will stop firing (delete the bindings first with 'versori notifications project unlink'
if you want a graceful tear-down).

If --channel-id is omitted, the CLI shows a picker of existing channels. Confirms before deleting
unless --yes is passed.

```sh
versori notifications channels delete --channel-id <id> [--yes] [flags]
```


**Flags:**
* `--channel-id`: ULID of the channel to delete (prompts a picker if omitted)
* `-h`, `--help`: help for delete
* `-y`, `--yes`: Skip the confirmation prompt



---

### `versori notifications channels list`




```sh
versori notifications channels list [flags]
```


**Flags:**
* `-h`, `--help`: help for list



---
