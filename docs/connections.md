---
title: "connections"
description: "Manage connections"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `create` | Create a new connection to a connection template |
| `delete` | Delete a connection |
| `list` | Lists connections for the current organisation context |
| `unlink` | Unlink a connection from an environment template |

---

### `versori connections create`


Create a new connection to a connection template. If an end user's external ID is provided,
The connection will be created as a dynamic connection for that end user, otherwise it will be created as a static connection.
If a base URL is not provided, it will default to the system's base URL defined in the connection template.

```sh
versori connections create --project <project-id> --environment <environment-name> --name <name> --template-id <template-id> [--external-id <external-id>] [--base-url <base-url>] [--<credential-field> <value>]... [flags]
```


**Flags:**
* `--api-key`: API key for authentication
* `--base-url`: Base URL for the connection, if not provided it will default to the systems base URL

* `--bypass`: Whether to bypass authentication (if supported by the connection template)

* `--client-id`: OAuth2 client id for use with an oauth2 client connection
* `--client-secret`: OAuth2 client secret for use with an oauth2 client connection

* `--env-file`: Path to .env file for resolving $VARIABLE references in credential flags

* `--environment`: The environment name within the project
* `--external-id`: External ID of the end user for the connection, if not provided the connection will be created as a static connection

* `-h`, `--help`: help for create
* `--name`: Name of the connection
* `--password`: Password for HTTP Basic authentication or OAuth2 password grant type

* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--template-id`: ID of the connection template to connect to
* `--token-url`: OAuth2 token URL for use with an oauth2 client connection. Defaults to the token URL defined in the connection template.

* `--username`: Username for HTTP Basic authentication or OAuth2 password grant type




---

### `versori connections delete`


Delete a connection (DELETE /o/\{organisation\}/connections/\{id\}).

This removes the connection itself. Unlink it from an environment first if you
only want to clear the active connection. Pass --yes or --confirm in non-interactive shells;
the VS Code extension always passes --yes after its own confirmation modal.

```sh
versori connections delete --id <connection-id> [--yes] [flags]
```


**Flags:**
* `--confirm`: Skip the confirmation prompt (same as --yes)
* `-h`, `--help`: help for delete
* `--id`: ULID of the connection to delete
* `-y`, `--yes`: Skip the confirmation prompt



---

### `versori connections list`




```sh
versori connections list [flags]
```


**Flags:**
* `--end-user`: Filter by end-user ULID or external ID (external IDs are resolved client-side)

* `-h`, `--help`: help for list
* `--system`: Filter by system ID



---

### `versori connections unlink`


Unlink a connection from an environment (DELETE /o/\{organisation\}/connections/\{id\}/link).

The connection itself is kept. Pass --yes or --confirm in non-interactive shells; the VS Code
extension always passes --yes after its own confirmation modal.

```sh
versori connections unlink --id <connection-id> --template-id <template-id> [--yes] [flags]
```


**Flags:**
* `--confirm`: Skip the confirmation prompt (same as --yes)
* `-h`, `--help`: help for unlink
* `--id`: ULID of the connection to unlink
* `--template-id`: Connection template ID (environment system) to unlink from
* `-y`, `--yes`: Skip the confirmation prompt



---
