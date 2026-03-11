---
title: "projects environments"
description: "Manage project environments"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `create` | Create a new environment by cloning an existing environment in the same project |
| `promote` | Promote one environment to another by syncing deployment configuration |
| `update-execution-pool` | Update the execution pool of an environment |

---

### `versori projects environments create`


Create a new environment by cloning an existing environment within a project.
This command creates a new environment by copying the details of an existing environment.
You can optionally clone systems and copy static user variables from the source environment.

```sh
versori projects environments create --project <project-id> --old-env <source-env> --new-env <new-env> [flags]
```


**Flags:**
* `--clone-systems`: If true, copies systems from the source environment to the new environment

* `--copy-static-user-variables`: If true, copies static user variables from the source environment to the new environment

* `--execution-pool`: Optional override execution pool for the new environment (defaults to source environment's execution pool)

* `-h`, `--help`: help for create
* `--new-env`: The name of the new environment to create
* `--old-env`: The name of the source environment to clone from
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects environments promote`


Promote one environment to another within a project.
This command copies the deployment configuration (currently the deployed docker image/version) 
from the source environment to the target environment.

For example, to promote staging to production:
  versori projects environments promote --project <id> --source staging --target production

```sh
versori projects environments promote --project <project-id> --source <source-env> --target <target-env> [flags]
```


**Flags:**
* `-h`, `--help`: help for promote
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--source`: The name of the source environment to promote from
* `--target`: The name of the target environment to promote to



---

### `versori projects environments update-execution-pool`


Update the execution pool used to deploy an environment.

WARNING: Changing the execution pool will result in a new public URL being created for the environment.
If the environment is running, it will be suspended first.

Example:
  versori projects environments update-execution-pool --environment 01ABC123 --execution-pool gcp

```sh
versori projects environments update-execution-pool --environment <environment-id> --execution-pool <execution-pool-name> [flags]
```


**Flags:**
* `--environment`: The name of the environment to update
* `--execution-pool`: The name of the new execution pool
* `-h`, `--help`: help for update-execution-pool
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `-y`, `--yes`: Skip confirmation prompt



---
