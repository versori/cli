---
title: "projects activations"
description: "Manage activations (end-user → environment links). Alias for the activation-lifecycle commands under `versori projects users`."
---

Manage activations on a project environment.

An activation links a previously-created end-user to a specific environment, supplying one
connection per environment system plus an optional bag of dynamic variables. This is the final
step in the per-end-user setup chain:

  1. versori users create -e \<id\> -n \<name\>                            (create end-user)
  2. versori connections create ... --external-id \<id\> ...             (create embedded connection)
  3. versori projects activations create ... --external-id \<id\> ...   (THIS COMMAND — link them)

The subcommands below are aliases for the equivalent 'versori projects users …' commands; use
whichever namespace feels more natural for the task at hand.

## Subcommands

| Subcommand | Description |
|---|---|
| `create` | Create an activation (link an end-user to an environment with its connections + variables) |
| `delete` | Delete an activation (unlinks an end-user from an environment) |
| `details` | Show one activation's details (connections + variables) |
| `list` | List end-users on a project environment with their activation status (active/inactive) |
| `set-variable` | Set a single dynamic variable on an activation |

---

### `versori projects activations create`


Activate an end-user on a project environment. The activation links an end-user to a
specific connection per environment system, plus an optional bag of dynamic variables that
workflow code reads via ctx.activation.getVariable('\<key\>').

The number of --connection flags must equal the number of environment systems for the target
environment (use 'versori projects systems list --project \<id\> --environment \<env\>' to enumerate them).
Omit --connection entirely to be prompted with a picker per environment system — the picker lists
the user's existing connections matching each system. Same for required dynamic variables: any
required schema entry not supplied via --variable / --variables-file is prompted interactively.

Dynamic variables can be supplied inline via repeatable --variable key=value flags, or in bulk
from a JSON file via --variables-file. Variables are validated against the project's
DynamicVariablesSchema (manage it with 'versori projects variables set'); unknown keys fail.

End-users themselves are created with 'versori users create -e \<external-id\> -n \<display-name\>'.
Connections are created with 'versori connections create' (use --external-id \<user\> for embedded
per-end-user connections). Once both exist, this command links them together.

```sh
versori projects activations create --project <project-id> --environment <environment-name> --external-id <user-external-id> [--connection <system-template-id>=<connection-id>]... [--variable key=value]... [--variables-file <path>] [flags]
```


**Flags:**
* `--connection`: Connection pair \<system-template-id\>=\<connection-id\> (repeatable; one per environment system). Omit to pick interactively.

* `--environment`: The environment name within the project
* `-e`, `--external-id`: External ID of the end-user to activate
* `-h`, `--help`: help for create
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--variable`: Dynamic variable in the form key=value (repeatable). Values are parsed as JSON when valid, else treated as strings. Missing required keys are prompted interactively.

* `--variables-file`: Path to a JSON file containing a flat object of dynamic variables (merged with --variable; --variable wins on conflicts)




---

### `versori projects activations delete`


Deactivate an end-user on a project environment by deleting the activation record.
The end-user themselves and any embedded connections are NOT deleted — only the link between
them and the environment. To re-activate the same end-user, run 'versori projects users activate'
again.

```sh
versori projects activations delete --project <project-id> --environment <environment-name> --external-id <user-external-id> [flags]
```


**Flags:**
* `--environment`: The environment name within the project
* `-e`, `--external-id`: External ID of the end-user to deactivate
* `-h`, `--help`: help for delete
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects activations details`




```sh
versori projects activations details --project <project-id> --environment <environment-name> --external-id <user-external-id> [flags]
```


**Flags:**
* `--environment`: The environment name within the project
* `--external-id`: The external ID of the user
* `-h`, `--help`: help for details
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects activations list`


List every end-user in the organisation alongside their activation status on the given
project environment. Active rows include the ActivationId and the activation's dynamic variables;
inactive rows show the end-user but leave activation-specific fields blank. Use -o yaml or
-o json to see the full row including DynamicVariables for active users.

```sh
versori projects activations list --project <project-id> --environment <environment-name> [flags]
```


**Flags:**
* `--environment`: The environment name within the project
* `-h`, `--help`: help for list
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects activations set-variable`


Set a single dynamic variable on an end-user's activation. The variable name must be
declared in the project's DynamicVariablesSchema first (manage it via 'versori projects variables
add'); unknown keys are pre-flighted locally and rejected before the request is sent.

The --value flag is parsed as JSON when valid (so '42', 'true', '"hello"', '\{"a":1\}' all work);
otherwise it is treated as a raw string. Variable updates take effect immediately at runtime —
no redeploy required.

```sh
versori projects activations set-variable --project <project-id> --environment <environment-name> --external-id <user-external-id> --name <variable-name> --value <value> [flags]
```


**Flags:**
* `--environment`: The environment name within the project
* `-e`, `--external-id`: External ID of the end-user
* `-h`, `--help`: help for set-variable
* `-n`, `--name`: Name of the variable to set
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--value`: Value for the variable (parsed as JSON when valid, else treated as a string)




---
