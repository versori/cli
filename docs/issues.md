---
title: "issues"
description: "List, inspect, and update issues"
---

Inspect and manage issues raised in the organisation.

Read commands (list, get) are safe to run for diagnosis. The update command changes live issue
state (e.g. acknowledging or resolving) and should only be run when explicitly requested.

## Subcommands

| Subcommand | Description |
|---|---|
| `get` | Get full detail for a single issue |
| `list` | List issues for the current organisation |
| `update` | Update an issue's status, resolution, or severity |

---

### `versori issues get`




```sh
versori issues get <issue-id> [flags]
```


**Flags:**
* `-h`, `--help`: help for get
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori issues list`


List issues raised in the organisation (via ctx.createIssue(), unhandled workflow errors, or
platform events such as an out-of-memory kill).

Filters (all optional) narrow server-side by status, project, environment, and severity. Inside a
synced project directory the project filter defaults from .versori. An OOM-killed project surfaces
here as an issue titled "OOM Killed" — inspect it with 'issues get \<id\>'.

```sh
versori issues list [flags]
```


**Flags:**
* `--after`: Pagination cursor: pass a prior response's last issue ID to fetch the next page

* `--before`: Pagination cursor for the previous page
* `--environment`: Filter by environment ID
* `--first`: Max issues to return (0 lets the server default apply)
* `-h`, `--help`: help for list
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--severity`: Filter by severity (critical, high, low, medium)
* `--status`: Filter by status (open, closed, acked, resolved)



---

### `versori issues update`


Update the editable fields of an issue. Common uses:

  issues update \<id\> --status acked                          # acknowledge
  issues update \<id\> --status resolved --resolution-status resolved

Only the flags you pass are changed; everything else is left as-is. This is a mutation — run it only
when explicitly asked.

```sh
versori issues update <issue-id> [flags]
```


**Flags:**
* `-h`, `--help`: help for update
* `--resolution-status`: Resolution status (resolved, negated, ignored)
* `--severity`: New severity (critical, high, low, medium)
* `--status`: New status (open, closed, acked, resolved)



---
