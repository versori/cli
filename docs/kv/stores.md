---
title: "kv stores"
description: "Inspect KV stores"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `list` | List KV stores in the current organisation |

---

### `versori kv stores list`


List the KV stores in the current organisation context. Stores are created lazily by the
runtime on first write and named ORG_\<org\>, PROJECT_\<project\>, or EXECUTION_\<project\>. Use the
returned store ID with the other kv commands' --store flag, or address a store by --scope.

```sh
versori kv stores list [flags]
```


**Flags:**
* `-h`, `--help`: help for list



---
