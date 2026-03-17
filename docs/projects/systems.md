---
title: "projects systems"
description: "Manage systems within a project"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `add` | Link a system to a project (create a connection template) |
| `bootstrap` | Create systems from research context |
| `delete-connection-template` | Delete a connection template from a project |
| `list-connections` | List static connections for a project environment |
| `list` | List systems linked to a project |
| `remove` | Remove a system (connection template) from a project |
| `update-connection-template` | Update a connection template in a project |

---

### `versori projects systems add`




```sh
versori projects systems add --project <project-id> --system <system-id> --name <name> --environment <environment-name> [--dynamic] [flags]
```


**Flags:**
* `--dynamic`: Whether the connection template is dynamic
* `--environment`: The environment name within the project
* `-h`, `--help`: help for add
* `--name`: A name for the connection template
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--system`: The system ID to link to the project



---

### `versori projects systems bootstrap`




```sh
versori projects systems bootstrap --file <path> [flags]
```


**Flags:**
* `-f`, `--file`: Path to a file containing research context (required)
* `-h`, `--help`: help for bootstrap
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--system-overrides`: JSON object of per-system overrides (e.g. '{"Stripe": {"base_url": "..."}}')




---

### `versori projects systems delete-connection-template`




```sh
versori projects systems delete-connection-template --project <project-id> --template <template-id> [flags]
```


**Flags:**
* `-h`, `--help`: help for delete-connection-template
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--template`: The connection template ID to delete



---

### `versori projects systems list-connections`




```sh
versori projects systems list-connections --project <project-id> --environment <environment-name> [flags]
```


**Flags:**
* `--environment`: The environment name within the project
* `-h`, `--help`: help for list-connections
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects systems list`




```sh
versori projects systems list --project <project-id> --environment <environment-name> [flags]
```


**Flags:**
* `--environment`: The environment name within the project
* `-h`, `--help`: help for list
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects systems remove`




```sh
versori projects systems remove --project <project-id> --template <connection-template-id> [flags]
```


**Flags:**
* `-h`, `--help`: help for remove
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--template`: The connection template ID to remove



---

### `versori projects systems update-connection-template`




```sh
versori projects systems update-connection-template --project <project-id> --template <template-id> [--name <name>] [--dynamic] [--auth-scheme-config-id <name>] [flags]
```


**Flags:**
* `--auth-scheme-config-id`: Auth scheme config ID to associate with the connection template

* `--dynamic`: Whether the connection template is dynamic
* `-h`, `--help`: help for update-connection-template
* `--name`: New name for the connection template
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--template`: The connection template ID to update



---
