---
title: "context"
description: "Manages the authentications to the Versori platform"
---

Context sets the configuration for your CLI session.
It allows you to manage your authentication tokens and organisations.

## Subcommands

| Subcommand | Description |
|---|---|
| `add` | Add a new context to your config and selects it as the default. It requires you generate a JWT token from the Versori console. You can generate the JWT here https://ai.versori.com/account?content=keys |
| `list` | List currently configured contexts. * denotes the active context. |
| `rm` | Remove a configured context. |
| `select` | Changes the active context. |
| `show` | Show the active context (without secrets). |

---

### `versori context add`




```sh
versori context add --name <name> --organisation <organisation-id> --jwt <jwt> [flags]
```


**Flags:**
* `-h`, `--help`: help for add
* `--jwt`: JWT token to use with the context. If the value is -, it will be read from stdin.

* `--name`: Name of the context
* `--organisation`: Organisation ID to use with the context



---

### `versori context list`




```sh
versori context list [flags]
```


**Flags:**
* `-h`, `--help`: help for list



---

### `versori context rm`




```sh
versori context rm [flags]
```


**Flags:**
* `-h`, `--help`: help for rm



---

### `versori context select`




```sh
versori context select <context-name> [flags]
```


**Flags:**
* `-h`, `--help`: help for select



---

### `versori context show`




```sh
versori context show [flags]
```


**Flags:**
* `-h`, `--help`: help for show



---
