---
title: "connections"
description: "Manage connections"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `create` | Create a new connection to a connection template |
| `list` | Lists connections for the current organisation context |

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

* `--environment`: The environment name within the project
* `--external-id`: External ID of the end user for the connection, if not provided the connection will be created as a static connection

* `-h`, `--help`: help for create
* `--name`: Name of the connection
* `--password`: Password for HTTP Basic authentication
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--template-id`: ID of the connection template to connect to
* `--username`: Username for HTTP Basic authentication



---

### `versori connections list`




```sh
versori connections list [flags]
```


**Flags:**
* `--end-user`: Filter by end user ID
* `-h`, `--help`: help for list
* `--system`: Filter by system ID



---
