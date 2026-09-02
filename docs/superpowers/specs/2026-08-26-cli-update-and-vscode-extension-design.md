# CLI self-update and VS Code / Cursor extension install

**Status:** Draft for review.  
**Date:** 2026-08-26  
**Repo:** `cli`  
**Does not change:** `install.sh` (curl one-liner stays CLI-binary-only).

Add `versori update` (replace this CLI binary, then refresh an already-installed editor extension) and `versori vscode install` (first-time / explicit extension install). Compatibility between CLI and vsix is a fetched JSON table, not matching version numbers.

---

## 1. What this builds

Two new commands. Curl install is unchanged.

| Command | CLI binary | Editor extension |
|---|---|---|
| `curl …/install.sh \| sh` | Install / overwrite `/usr/local/bin/versori` | Never |
| `versori update` | Overwrite **this** binary (latest, or `--version`) | Only if that editor already has `versori.versori-vscode`; then install the latest **compatible** vsix (upgrade or downgrade) |
| `versori vscode install` | Never | Always install the latest compatible vsix into the chosen editor(s) |

`install.sh` is how people who do not yet have `versori update` get a newer CLI. Do not add prompts or vsix steps there.

---

## 2. Config isolation (must not touch)

Neither command reads or writes:

- `~/.versori/config.yaml` or any other file under `~/.versori/`
- project `.versori` pins
- VS Code / Cursor user config (`settings.json`, keybindings, snippets, `argv.json`)

Allowed writes:

- the `versori` executable (update only)
- a temp `.vsix` (deleted afterward)
- whatever `code --install-extension` / `cursor --install-extension` writes under that editor’s **extensions** directory

Do not invent `~/.versori/extensions` or symlinks into `~/.vscode/extensions` / `~/.cursor/extensions`.

---

## 3. Compatibility table

Canonical file in this repo (fetched at runtime from `main`, not baked into the binary):

`https://raw.githubusercontent.com/versori/cli/main/vscode-compat.json`

Shape:

```json
{
  "extension_id": "versori.versori-vscode",
  "releases": [
    { "vsix": "0.1.0", "min_cli": "0.1.0" }
  ]
}
```

- `extension_id` is `publisher.name` from the extension `package.json` (`versori` + `versori-vscode`).
- `vsix` is the GitHub release tag on `versori/versori-vscode-extension` (strip a leading `v` for compare and for the download URL).
- `min_cli` is the oldest CLI that vsix may run against (strip a leading `v`). Semver compare. Numbers do not have to match; 0.1.0 aligning is coincidence.
- Editing this file on `main` changes policy without a new CLI or vsix build.

### Picker

Pure function, fixture-tested. Inputs: CLI version string, parsed table. Output: vsix tag to install, or “none compatible”.

1. Fail closed if the table cannot be fetched or parsed (`vscode install` exits non-zero; `update` warns and skips vsix).
2. Keep rows where `min_cli <=` the **CLI version used for this decision** (see below).
3. Among those, pick the **newest** `vsix`.
4. If none: do not install. Message that this CLI is too old for any published extension, and tell them to run `versori update`, or:

   `curl -fsSL https://raw.githubusercontent.com/versori/cli/main/install.sh | sh`

   if they may not have `update` yet.

| CLI version used | Table | Result |
|---|---|---|
| `0.0.9` | vsix `0.1.0` / `min_cli` `0.1.0` | None. Too-old message. |
| `0.1.0` | same | vsix `0.1.0` |
| `0.1.1` | same | vsix `0.1.0` |
| `0.1.0` after later vsix `0.4.0` with `min_cli` `0.3.0` | still vsix `0.1.0` | Newest whose min still fits |

**Which CLI version is “this decision”:**

- `versori vscode install`: this process’s version (`pkg/cmd.version` / `versori version`).
- `versori update`: the **target** version being installed (latest tag, or `--version`), **not** the in-memory version of the process that started. After overwrite the running process still reports the old version.

If the extension is already installed and the target CLI is older, the pick may be an older vsix. Install it anyway (downgrade). Internal editor `--force` so VS Code/Cursor replace the newer copy.

---

## 4. `versori update`

Root command. Not under `projects`.

### CLI binary

1. Resolve the running binary: `os.Executable()` then `EvalSymlinks`. Update that path, not always `/usr/local/bin`.
2. Resolve target version: `--version vX.Y.Z` if set, else GitHub `versori/cli` latest release tag (same source as `install.sh`).
3. If the file on disk is already that version, skip the download; still run the vsix pass.
4. Download `cli_<ver>_<os>_<arch>.tar.gz` and `checksums.txt` from that release. SHA-256 must match. Same filename rules as `install.sh` (strip leading `v` for the tarball name; keep the tag in the URL).
5. Write beside the current binary, `chmod +x`, rename over it.
6. If the directory is not writable: `sudo` the move onto the **already-resolved path** (do not `sudo versori update`; sudo PATH may be a different binary). If sudo is refused or missing, exit non-zero with that curl one-liner. Do not leave a half-written binary.

Darwin and Linux only for the binary replace (same as `install.sh`). Windows: print that Windows should download a release zip; do not attempt self-update. Vsix pass may still run on Windows if `code`/`cursor` resolve.

### Vsix pass (after binary step)

For each resolved editor:

1. `<bin> --list-extensions`. If stdout does not contain `versori.versori-vscode`, skip that editor. No prompt.
2. If present: picker using **target** CLI version; download that vsix; `<bin> --install-extension <vsix> --force`.
3. Any list/download/install failure: **warn on stderr, do not fail** the command if the CLI binary step succeeded (or was already current).

No Y/N for the extension on `update`.

---

## 5. `versori vscode install`

Parent `versori vscode`, subcommand `install`. No `--force` flag. A successful run always installs (first time or replace). Internally the editor CLI still gets `--force` so a re-run or downgrade actually replaces.

Missing vsix release, bad checksum, or empty picker: **exit non-zero**.

### Editor binaries

Flags (win if set):

- `--vscode-path` — absolute path to the VS Code CLI (`code`), not the `.app` and not the extensions folder
- `--cursor-path` — absolute path to the Cursor CLI (`cursor`)

If a flag is set and that path is missing or not executable: `vscode install` errors for that editor. `update` warns and skips that editor.

If omitted, search in order:

1. `code` / `cursor` on `PATH`
2. Well-known locations:

| Editor | Darwin | Linux | Windows |
|---|---|---|---|
| VS Code | `/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code` then Insiders equivalent | `/usr/bin/code`, `/usr/share/code/bin/code` | `%LOCALAPPDATA%\Programs\Microsoft VS Code\bin\code.cmd` |
| Cursor | `/Applications/Cursor.app/Contents/Resources/app/bin/cursor` | `/usr/bin/cursor`, `/usr/share/cursor/bin/cursor` | `%LOCALAPPDATA%\Programs\cursor\resources\app\bin\cursor.cmd` |

Unresolved editor = not installed. No prompt for it.

### Prompts (huh, same as other CLI confirms)

- **One** resolved editor: install immediately. Running `versori vscode install`
  is explicit consent, so a second confirmation is redundant.
- **Both**: select one of: VS Code, Cursor, Both, Cancel.
- **None**: say neither was found; mention `--vscode-path` / `--cursor-path`; exit non-zero.

Non-TTY: with one resolved editor, install immediately. With both resolved
editors, do not run huh (it blocks): without `--yes` / `--confirm` (existing
skip-prompt aliases), print that a TTY or `--yes` is required to choose an
editor and exit non-zero. With `--yes`, install into every resolved editor
(path flags still limit which binaries exist). `--yes` only skips the
multi-editor choice; it is not a versori `--force`.

### Vsix download

From `versori/versori-vscode-extension` GitHub Releases, tag = picker `vsix` (prefix `v` if the release tags are `v0.1.0`). Asset: the `.vsix` (prefer `versori-vscode-*.vsix`). SHA-256 against `checksums.txt` on that **same** release. No checksum → do not install.

Those assets must be downloadable **without a GitHub token** (public repo or public release assets). Temp file in `os.TempDir`, delete after.

---

## 6. Errors and messaging

| Situation | `versori update` | `versori vscode install` |
|---|---|---|
| Compat table fetch/parse fail | Warn; skip vsix | Non-zero |
| No compatible vsix | Warn + update/curl hint; skip vsix | Non-zero + same hint |
| Editor `--install-extension` fail | Warn; CLI success still counts | Non-zero |
| CLI tarball checksum fail | Non-zero; no vsix | n/a |
| No editors resolved | One line that no VS Code/Cursor CLI was found; CLI success still counts | Non-zero + path-flag hint |

Do not print JWTs, config paths as if we wrote them, or extension-folder internals.

---

## 7. Docs

- Cobra `Short`/`Long`/`Use` on the new commands (this repo’s `docs/*.md` is generated by `make generate`).
- `README.md` installation section: curl still the first install; `versori update` for later CLI updates; `versori vscode install` for the editor.
- `user-docs/latest/cli/installation.mdx` the same (via `make versori-docs` or a matching edit).
- Do not put this in `coding-versori-sdk` except a one-line pointer if generated CLI docs already cover it. Agents building workflows do not need to install the editor extension.

Ship an initial `vscode-compat.json` on `main` with the first row `{ "vsix": "0.1.0", "min_cli": "0.1.0" }` when the first vsix release exists. Until that release is public, `vscode install` correctly fails closed.

---

## 8. Tests

No live GitHub or live `code` in unit tests.

- Picker: table-driven (the examples in §3, plus empty table, equal min_cli, newer vsix with higher min_cli left unpicked).
- Target-version vs process-version: `update` passes the target string into the picker.
- Editor selection: one resolved editor installs without a prompt, including
  in a non-TTY; both still require the chooser or `--yes` / `--confirm`.
- Config isolation: commands under test must not create or truncate `~/.versori/config.yaml` (use a fake home / do not call `LoadConfigAndContext` unless needed; these commands do not need org context).

---

## Handoff for writing-plans

The planner has this file and the repo, not the design conversation.

**Locked decisions:**
- `install.sh` stays CLI-only — curl is first install and the fallback updater; no vsix, no prompt
- `versori update` replaces the resolved running binary, then vsix only if already installed — no extension prompt
- `versori vscode install` always installs; no user `--force`; editor CLI still gets `--force` internally
- Compat lives in `vscode-compat.json` on `cli` `main`, fetched at runtime — not hardcoded; versions need not match numerically
- Picker: newest vsix whose `min_cli <=` decision CLI version; empty → too-old message naming `versori update` and the curl one-liner
- `update` picker uses **target** CLI version, not in-memory `version` after overwrite — so pinning an older CLI downgrades the vsix if installed
- `--vscode-path` / `--cursor-path` override; else PATH then the well-known table in §5
- Prompts: one editor installs immediately; both → VS Code / Cursor / Both /
  Cancel; a non-TTY with both requires `--yes`/`--confirm`
- Vsix from `versori/versori-vscode-extension` releases + `checksums.txt`; public download
- Fail-soft vsix on `update`; fail-hard on `vscode install`
- Never write `~/.versori/**`, project `.versori`, or editor user settings
- Binary self-update: darwin/linux only; sudo moves onto the resolved path only
- `--version` on `update` pins the CLI release tag (same idea as `VERSORI_VERSION` on curl)
- No vsix pin flag — the table always chooses

**Rejected alternatives:**
- Prompt or vsix inside `curl \| sh` — user: curl is the updater, keep it simple
- Symlink vsix tree from `~/.versori` into editor extensions folders — fights versioned extension dirs; would mix with CLI config
- Marketplace / Open VSX id install — user publishes GitHub vsix
- Vsix attached to CLI GitHub releases — user chose the extension repo
- `versori update` as the only extension path — also need explicit `vscode install`
- User-facing `--force` on `vscode install` — always install instead
- Baking min_cli into the binary — not updateable without a CLI release
- Replacing always `/usr/local/bin/versori` — GOPATH/`~/go/bin` installs would be ignored
- `sudo versori update` as the permission story — sudo PATH can pick another binary

**Repo context the planner must use:**
- Read first: `install.sh` — tarball name, checksums, `/usr/local/bin` + sudo mv
- Read first: `pkg/cmd/root.go` — how root commands are registered
- Read first: `pkg/cmd/skills.go` + `pkg/cmd/skills/download.go` — parent + subcommand layout; unauthenticated GitHub HTTP
- Pattern to copy: `pkg/cmd/flags/skip_prompt.go` — `--yes` / `--confirm`
- Pattern to copy: `pkg/cmd/elements/confirm.go` and `listselect.go` — huh prompts
- Pattern to copy: `.goreleaser.yaml` ldflags `pkg/cmd.version` — released version string
- Do not change: `install.sh` behaviour — curl stays binary-only
- Do not change: `pkg/cmd/config/config.go` load/save of `~/.versori/config.yaml` — these commands must not call it unless a later flag needs it (they do not)
- Do not change: generated `docs/*.md` by hand — set cobra help, then `make generate`

**Interfaces / data already decided:**
- `vscode-compat.json`: `{ "extension_id": "versori.versori-vscode", "releases": [{ "vsix": "<semver>", "min_cli": "<semver>" }] }`
- Picker: `(cliVersion string, table CompatTable) → (vsixTag string, ok bool)`
- `versori update [--version <tag>] [--vscode-path <path>] [--cursor-path <path>]`
- `versori vscode install [--vscode-path <path>] [--cursor-path <path>] [--yes|--confirm]`
- Editor id string: `versori.versori-vscode`
- Extension GitHub repo: `versori/versori-vscode-extension`
- CLI GitHub repo: `versori/cli`

**Constraints (verbatim):**
- One resolved editor installs without confirmation
- `--yes` / `--confirm` skip the two-editor chooser only
- Checksum required for CLI tarball and vsix
- Initial table row when first vsix exists: `"vsix": "0.1.0", "min_cli": "0.1.0"`
- Curl fallback: `curl -fsSL https://raw.githubusercontent.com/versori/cli/main/install.sh | sh`
- Compat URL: `https://raw.githubusercontent.com/versori/cli/main/vscode-compat.json`

**Out of scope:**
- Changing `install.sh` — stays as today
- Publishing the vsix / making the extension repo public — human; commands fail closed until assets exist
- Windows CLI self-update — zip from releases, same as today
- VS Marketplace / Open VSX
- Homebrew formula
- Symlinks or a shared extension directory under `~/.versori`
- `coding-versori-sdk` workflow research/deploy behaviour

**Done looks like:**
- `versori update --help` and `versori vscode install --help` exist; `install.sh` diff empty of vsix/prompt
- Unit tests cover picker examples in §3 and target-version vs process-version
- `vscode install` with one resolved editor installs without prompting, even in
  a fake non-TTY
- `vscode install` with both resolved editors in a fake non-TTY without `--yes`
  exits non-zero without calling either editor
- Commands under test do not write `config.yaml`
- `make generate` reflects the new commands
- A developer can add a vsix/`min_cli` row to `vscode-compat.json` without shipping a new CLI
