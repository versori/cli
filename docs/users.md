---
title: "users"
description: "Manage users"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `create` | Create a new user in the current organisation |
| `delete` | Delete an end-user from the current organisation |
| `list` | Lists users in the current organisation context |

---

### `versori users create`




```sh
versori users create --display-name <name> --external-id <id> [flags]
```


**Flags:**
* `-n`, `--display-name`: Display name of the user
* `-e`, `--external-id`: External ID of the user
* `-h`, `--help`: help for create



---

### `versori users delete`


Delete an end-user from the current organisation (DELETE /o/\{organisation\}/users/\{user_id\}).

This removes the end-user record itself, not just an activation on one environment. Activations
and embedded connections owned by that user are also removed by the platform.

Pass --id (the user ULID) or --external-id (resolved client-side). Confirms before deleting
unless --yes or --confirm is passed; in a non-interactive shell one of those flags is required.

```sh
versori users delete (--id <ulid> | --external-id <id>) [--yes] [flags]
```


**Flags:**
* `--confirm`: Skip the confirmation prompt (same as --yes)
* `-e`, `--external-id`: External ID of the end-user to delete (resolved to a ULID)
* `-h`, `--help`: help for delete
* `--id`: ULID of the end-user to delete
* `-y`, `--yes`: Skip the confirmation prompt



---

### `versori users list`




```sh
versori users list [flags]
```


**Flags:**
* `-h`, `--help`: help for list



---
