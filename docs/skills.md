---
title: "skills"
description: "Manages the skills"
---

Skills sets the configuration for your CLI session.
It allows you to manage your skills.

## Subcommands

| Subcommand | Description |
|---|---|
| `download` | Download AI agent skills |

---

### `versori skills download`


Download the AI agent skills to a local directory.

By default, this command extracts the skills that are shipped with the current CLI version,
requiring no network connection.

If the --latest flag is provided, it will attempt to download the latest version of the
skills directly from the main branch of the GitHub repository.

```sh
versori skills download [flags]
```


**Flags:**
* `--agent`: Combine skills into a single AGENTS.md file
* `-d`, `--directory`: Directory to save the skills into. It will append the skill name to the directory.

* `-h`, `--help`: help for download
* `--latest`: Download the latest version of the skills



---
