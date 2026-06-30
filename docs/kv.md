---
title: "kv"
description: "Inspect and manage KV store entries"
---

Inspect and manage entries in a project's KV store.

Read commands (stores list, list, count, get) are safe to run for diagnosis. Mutation commands
(set, delete, wipe) change live workflow state and should only be run when explicitly requested.

## Subcommands

| Subcommand | Description |
|---|---|
| `count` | Count KV entries matching a prefix |
| `delete` | Delete a single KV entry (mutation) |
| `get` | Get a single KV entry by key |
| `list` | List KV entries under a prefix |
| `set` | Set a KV entry (mutation) |
| `stores` | Inspect KV stores |
| `wipe` | Delete every KV entry under a prefix (bulk mutation) |

---

### `versori kv count`


Count the entries matching a key prefix. Unlike list, this returns a total rather than a
capped page, so it is the right way to answer "how many items are under this prefix?".

Targeting works the same as kv list: --store \<id\> (raw) or --scope \<scope\> (friendly), with
optional --prefix segments to narrow further.

```sh
versori kv count (--store <id> | --scope <scope> ...) [--prefix <segment>]... [flags]
```


**Flags:**
* `--activation-id`: Activation ID (optional; narrows project/execution scope).
* `--environment`: Environment name, e.g. production (friendly mode).
* `--execution-id`: Execution ID (required for --scope execution).
* `--external-id`: End-user external ID (required for --scope user).
* `--full-keys`: Show full resolved keys instead of stripping the scope prefix.

* `-h`, `--help`: help for count
* `--prefix`: Key prefix segment (repeatable).
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--scope`: Friendly scope: organization | workspace | project | user | execution.

* `--store`: KV store ID (raw mode). Mutually exclusive with --scope.



---

### `versori kv delete`


Delete one entry by key.

WARNING: this mutates live workflow state. Only run it when explicitly asked to delete a specific
key — never as a side-effect of debugging. This removes a single key only; to delete everything
under a prefix use kv wipe.

Confirms before deleting unless --yes is passed; in non-interactive shells --yes is required.

```sh
versori kv delete (--store <id> | --scope <scope> ...) --key <a/b/c> [flags]
```


**Flags:**
* `--activation-id`: Activation ID (optional; narrows project/execution scope).
* `--environment`: Environment name, e.g. production (friendly mode).
* `--execution-id`: Execution ID (required for --scope execution).
* `--external-id`: End-user external ID (required for --scope user).
* `--full-keys`: Show full resolved keys instead of stripping the scope prefix.

* `-h`, `--help`: help for delete
* `--key`: Key to delete (slash-delimited).
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--scope`: Friendly scope: organization | workspace | project | user | execution.

* `--store`: KV store ID (raw mode). Mutually exclusive with --scope.
* `-y`, `--yes`: Skip the confirmation prompt.



---

### `versori kv get`


Fetch one entry by its key. In raw mode (--store) the --key is the literal full key. In
friendly mode (--scope) the --key is relative to the scope and the CLI prepends the scope prefix.

Values written by workflows are JSON-encoded strings and are unwrapped one level by default; pass
--raw-values to show exactly what is stored.

```sh
versori kv get (--store <id> | --scope <scope> ...) --key <a/b/c> [flags]
```


**Flags:**
* `--activation-id`: Activation ID (optional; narrows project/execution scope).
* `--environment`: Environment name, e.g. production (friendly mode).
* `--execution-id`: Execution ID (required for --scope execution).
* `--external-id`: End-user external ID (required for --scope user).
* `--full-keys`: Show full resolved keys instead of stripping the scope prefix.

* `-h`, `--help`: help for get
* `--key`: Key to fetch (slash-delimited, e.g. cursor/orders).
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--raw-values`: Show the value exactly as stored (no JSON-string unwrapping).

* `--scope`: Friendly scope: organization | workspace | project | user | execution.

* `--store`: KV store ID (raw mode). Mutually exclusive with --scope.



---

### `versori kv list`


List key/value entries in a store, optionally narrowed by a key prefix.

Targeting (pick one):
  --store \<id\>     raw mode; --prefix segments are the literal key prefix.
  --scope \<scope\>  friendly mode; the CLI derives the store + scope prefix and strips it from
                   displayed keys (use --full-keys to keep it). Extra --prefix segments narrow
                   within the scope.

Values written by workflows are JSON-encoded strings; by default they are unwrapped one level for
readability. Pass --raw-values to show exactly what is stored. For nested values prefer -o json or
-o yaml, which preserve the full structure (table truncates the value column).

Beyond the prefix, entries can be filtered server-side by creation time (--created-after /
--created-before) and by metadata (--metadata key=value).

Pagination is cursor-based, ordered newest-first by the entry's internal ID (not by key). A page
is capped (max 100). To page through results, pass the previous response's opaque nextCursor to
--after — never a key. An empty nextCursor means you have reached the end; it does not mean more
pages are hidden. Use 'kv count' for an authoritative total under a prefix.

```sh
versori kv list (--store <id> | --scope <scope> ...) [--prefix <segment>]... [flags]
```


**Flags:**
* `--activation-id`: Activation ID (optional; narrows project/execution scope).
* `--after`: Pagination cursor for the next page: pass the previous response's opaque nextCursor verbatim (NOT a key). Empty nextCursor means there are no more entries.

* `--before`: Pagination cursor for the previous page: pass an opaque prevCursor from a prior response (NOT a key).

* `--created-after`: Only entries created at/after this time (RFC3339 or YYYY-MM-DD).

* `--created-before`: Only entries created at/before this time (RFC3339 or YYYY-MM-DD).

* `--environment`: Environment name, e.g. production (friendly mode).
* `--execution-id`: Execution ID (required for --scope execution).
* `--external-id`: End-user external ID (required for --scope user).
* `--full-keys`: Show full resolved keys instead of stripping the scope prefix.

* `-h`, `--help`: help for list
* `--limit`: Max entries to return (1-100; 0 lets the server default apply).

* `--metadata`: Filter by metadata key=value (repeatable; value parsed as JSON when valid, else a string).

* `--prefix`: Key prefix segment (repeatable).
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--raw-values`: Show values exactly as stored (no JSON-string unwrapping).
* `--reverse`: Reverse the sort order.
* `--scope`: Friendly scope: organization | workspace | project | user | execution.

* `--store`: KV store ID (raw mode). Mutually exclusive with --scope.



---

### `versori kv set`


Write a value at a key.

WARNING: this mutates live workflow state (cursors, dedupe keys, batch progress). Only run it when
explicitly asked to set a specific key — never as a side-effect of debugging.

The value is JSON-encoded the same way the runtime SDK writes values, so workflow code that reads
the key with ctx.openKv().get() round-trips correctly. A --value that parses as JSON keeps its type
(object/array/number/bool); otherwise it is stored as a string.

Confirms before writing unless --yes is passed; in non-interactive shells --yes is required.

```sh
versori kv set (--store <id> | --scope <scope> ...) --key <a/b/c> (--value <json> | --value-file <path>) [flags]
```


**Flags:**
* `--activation-id`: Activation ID (optional; narrows project/execution scope).
* `--environment`: Environment name, e.g. production (friendly mode).
* `--execution-id`: Execution ID (required for --scope execution).
* `--expire-in`: TTL in milliseconds (0 = no expiry).
* `--external-id`: End-user external ID (required for --scope user).
* `--full-keys`: Show full resolved keys instead of stripping the scope prefix.

* `-h`, `--help`: help for set
* `--if-not-exists`: Only set if the key does not already exist.
* `--key`: Key to set (slash-delimited).
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--scope`: Friendly scope: organization | workspace | project | user | execution.

* `--store`: KV store ID (raw mode). Mutually exclusive with --scope.
* `--value`: Value to store (JSON when valid, otherwise a string).
* `--value-file`: Read the value from a file instead of --value.
* `-y`, `--yes`: Skip the confirmation prompt.



---

### `versori kv stores`



See the [stores reference](/docs/kv/stores.md) for detailed subcommands to inspect kv stores.


---

### `versori kv wipe`


Cascade-delete all entries under a key prefix.

WARNING: this is the most destructive KV operation and mutates live workflow state in bulk. Only
run it when explicitly asked to wipe a specific prefix or scope — never as a side-effect of
debugging or to "clean up".

An empty prefix is refused (that would target the entire store). The command always counts the
matching entries first and prints what would be deleted; it only proceeds when --confirm is passed,
so running it without --confirm is a safe dry-run.

```sh
versori kv wipe (--store <id> --prefix <segment>... | --scope <scope> ...) --confirm [flags]
```


**Flags:**
* `--activation-id`: Activation ID (optional; narrows project/execution scope).
* `--confirm`: Actually perform the deletion (without it, prints a dry-run preview).

* `--environment`: Environment name, e.g. production (friendly mode).
* `--execution-id`: Execution ID (required for --scope execution).
* `--external-id`: End-user external ID (required for --scope user).
* `--full-keys`: Show full resolved keys instead of stripping the scope prefix.

* `-h`, `--help`: help for wipe
* `--prefix`: Key prefix segment (repeatable). Required in raw mode.
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `--scope`: Friendly scope: organization | workspace | project | user | execution.

* `--store`: KV store ID (raw mode). Mutually exclusive with --scope.



---
