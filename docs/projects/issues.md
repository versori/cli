---
title: "projects issues"
description: "List, inspect, and update issues for a project"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `get` | Get full detail for a single issue |
| `list` | List issues for a project |
| `update` | Update an issue's status or severity |

---

### `versori projects issues get`




```sh
versori projects issues get <issue-id> [flags]
```


**Flags:**
* `--environment`: Environment name to filter by (auto-selected when the project has exactly one environment)

* `-h`, `--help`: help for get
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects issues list`


List issues raised for a single project (via ctx.createIssue(), unhandled workflow errors, or
platform events such as an out-of-memory kill).

--project is required; it defaults from .versori inside a synced project directory. --environment is
an environment name resolved against the project, and is auto-selected when the project has exactly
one environment.

```sh
versori projects issues list [flags]
```


**Flags:**
* `--after`: Pagination cursor: pass a prior response's last issue ID to fetch the next page

* `--before`: Pagination cursor for the previous page
* `--environment`: Environment name to filter by (auto-selected when the project has exactly one environment)

* `--first`: Max issues to return (0 lets the server default apply)
* `-h`, `--help`: help for list
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--severity`: Filter by severity (critical, high, low, medium)
* `--status`: Filter by status (open, closed, acked, resolved)



---

### `versori projects issues update`


Update the editable fields of an issue. Common uses:

  update \<id\> --status acked      # acknowledge
  update \<id\> --status resolved   # mark resolved (server records resolved_at)

Mirrors the platform: only the fields you pass are sent. This is a mutation — run it only when
explicitly asked.

```sh
versori projects issues update <issue-id> [flags]
```


**Flags:**
* `-h`, `--help`: help for update
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--severity`: New severity (critical, high, low, medium)
* `--status`: New status (open, closed, acked, resolved)



---
