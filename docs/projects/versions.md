---
title: "projects versions"
description: "Manage versions within a project."
---



## Subcommands

| Subcommand | Description |
|---|---|
| `create` | Creates a new version for a project with the files from --directory |
| `deploy` | Creates a new version for a project with the files from --directory |
| `list` | Lists most recent versions |
| `pull` | Pull files from a version into the target directory |

---

### `versori projects versions create`




```sh
versori projects versions create [--project <project-id>] [--directory <source_directory> [--name <version-name>] [--description <version-description>] [--dry-run] [flags]
```


**Flags:**
* `--description`: Description of the new version.
* `-d`, `--directory`: Directory containing the new version.
* `--dry-run`: Print files that would be uploaded without actually pushing.
* `-h`, `--help`: help for create
* `-n`, `--name`: Name of the new version.
* `--project`: ID of the project to push the version to.



---

### `versori projects versions deploy`




```sh
versori projects versions deploy --project <project-id> --version <version-id> --environment <environment-name> [flags]
```


**Flags:**
* `--environment`: The name of the environment to deploy to.
* `-h`, `--help`: help for deploy
* `--project`: The ID of the project to deploy the version to.
* `--version-id`: The ID of the version to deploy.



---

### `versori projects versions list`




```sh
versori projects versions list [flags]
```


**Flags:**
* `-h`, `--help`: help for list
* `--limit`: How many versions to list
* `--project`: The ID of the project to list versions for.



---

### `versori projects versions pull`




```sh
versori projects versions pull [--directory <target-directory>] [--project <project-id>] [--version <version-id>] [--dry-run] [flags]
```


**Flags:**
* `-d`, `--directory`: Directory containing the versions to be uploaded
* `--dry-run`: Print files that would be uploaded without actually pushing.
* `-h`, `--help`: help for pull
* `--project`: ID of the project to push the version to.
* `--version`: ID of the version. If not given, you will be prompted to select one.




---
