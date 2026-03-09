---
title: "projects"
description: "Manage projects, deployments, versions, assets, and environments"
---

The `projects` command (alias: `project`) provides subcommands for the full project lifecycle — creation, file management, deployment, logging, and more.


## Subcommands

| Subcommand | Description |
|---|---|
| `asset` | Manage assets within a project |
| `create` | Create a new project in the current organisation |
| `deploy` | Deploy the project to versori |
| `details` | Get details for a project. Pass in - to read the project id from stdin |
| `edit` | Edit project environment configuration (resource limits and requests) |
| `environments` | Manage project environments |
| `list` | Lists all projects in the current context |
| `logs` | Check the project logs |
| `proxy` | Send an HTTP request to a project's deployed environment |
| `save` | Save the project files to versori (no deploy) |
| `sync` | Sync pulls the project files to the local directory. WARNING: This will overwrite any local changes |
| `systems` | Manage systems within a project |
| `users` | Manage users (activations) within a project |
| `versions` | Manage versions within a project. |

---

### `versori projects asset`



See the [asset reference](/docs/projects/asset.md) for detailed subcommands to manage assets within a project.


---

### `versori projects create`




```sh
versori projects create --name <name> [flags]
```


**Flags:**
* `-h`, `--help`: help for create
* `-n`, `--name`: Name of the project



---

### `versori projects deploy`




```sh
versori projects deploy [--project <project-id>] --environment <env> [--directory <directory>] [flags]
```


**Flags:**
* `--assets`: Also upload assets from the versori-research directory
* `--description`: Description of the new version.
* `-d`, `--directory`: The directory containing the project files
* `--dry-run`: Print files that would be uploaded without actually deploying

* `--environment`: The project environment to deploy to (e.g. production, staging)

* `-h`, `--help`: help for deploy
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--version`: Name of the version to create (default: current time in UTC, e.g. 2006-01-02-15-04-05)




---

### `versori projects details`




```sh
versori projects details <project-id> [flags]
```


**Flags:**
* `-h`, `--help`: help for details



---

### `versori projects edit`




```sh
versori projects edit --project <project-id> --environment <environment-name> [flags]
```


**Flags:**
* `--environment`: The environment name
* `-h`, `--help`: help for edit
* `--max-replicas`: Maximum number of replicas. Setting this option enables autoscaling on the project

* `--project`: The project ID
* `--replicas`: Number of replicas
* `--resource.cpu.limits`: CPU limits (e.g., 500m)
* `--resource.cpu.requests`: CPU requests (e.g., 100m)
* `--resource.memory.limits`: Memory limits (e.g., 500Mi)
* `--resource.memory.requests`: Memory requests (e.g., 200Mi)
* `--service-account`: Service account to use for the environment. Pass an empty string to remove the service account

* `--static-ip`: Enable or disable static IP (enabled/disabled)



---

### `versori projects environments`



See the [environments reference](/docs/projects/environments.md) for detailed subcommands to manage project environments.


---

### `versori projects list`




```sh
versori projects list [flags]
```


**Flags:**
* `-h`, `--help`: help for list



---

### `versori projects logs`




```sh
versori projects logs  [flags]
```


**Flags:**
* `--environment`: The environment to retrieve logs for. e.g. (production, staging)

* `-h`, `--help`: help for logs
* `--limit`: How many logs to retrieve; 0 means no explicit limit
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--search`: Search query to filter logs
* `--since`: Go duration since now, e.g. 24h, 2h30m (default: 24h)



---

### `versori projects proxy`


Proxy sends an HTTP request to a project's deployed environment and returns
the response. This allows triggering a project remotely via an HTTP call.

The --body flag accepts a raw string or @filename to read the body from a file.
When using @filename, the Content-Type header is automatically inferred from
the file extension unless explicitly provided via --header.

```sh
versori projects proxy --path <path> [flags]
```


**Flags:**
* `-b`, `--body`: Request body (string or @filename to read from file)
* `--environment`: Project environment name (e.g. production, staging)
* `-H`, `--header`: HTTP headers as key:value pairs (repeatable)
* `-h`, `--help`: help for proxy
* `-m`, `--method`: HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
* `--path`: URL path to call on the integration (required)
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `-q`, `--query`: Query parameters as key:value pairs (repeatable)



---

### `versori projects save`




```sh
versori projects save [--project <project-id>] [--directory <directory>] [flags]
```


**Flags:**
* `--assets`: Also upload assets from the versori-research directory
* `-d`, `--directory`: The directory containing the project files
* `--dry-run`: Print files that would be uploaded without actually saving
* `-h`, `--help`: help for save
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects sync`


Sync pulls the project files to the local directory. The --project flag is only required the first time you sync a project.
WARNING: This will overwrite any local changes

```sh
versori projects sync [--project <project-id>] [--directory <directory>]
```


**Flags:**
* `--assets`: Also sync project assets, removing any that are no longer part of the project from the versori-research directory

* `-d`, `--directory`: The directory to download the project files into
* `--dry-run`: Print files that would be created/updated/deleted without actually syncing

* `-h`, `--help`: help for sync
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects systems`



See the [systems reference](/docs/projects/systems.md) for detailed subcommands to manage systems within a project.


---

### `versori projects users`



See the [users reference](/docs/projects/users.md) for detailed subcommands to manage users (activations) within a project.


---

### `versori projects versions`



See the [versions reference](/docs/projects/versions.md) for detailed subcommands to manage versions within a project..


---
