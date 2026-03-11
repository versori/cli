---
title: "projects users"
description: "Manage users (activations) within a project"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `details` | Get the details for an individual end user of a project |
| `list` | List users (activations) linked to a project environment |

---

### `versori projects users details`




```sh
versori projects users details --project <project-id> --environment <environment-name> --external-id <user-id> [flags]
```


**Flags:**
* `--environment`: The environment name within the project
* `--external-id`: The external ID of the user
* `-h`, `--help`: help for details
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects users list`




```sh
versori projects users list --project <project-id> --environment <environment-name> [flags]
```


**Flags:**
* `--environment`: The environment name within the project
* `-h`, `--help`: help for list
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---
