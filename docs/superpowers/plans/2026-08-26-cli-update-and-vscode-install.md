# CLI self-update and VS Code extension install Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `versori update` (replace this CLI binary, then refresh an already-installed editor extension) and `versori vscode install` (explicit vsix install), with a fetched CLI↔vsix compatibility table.

**Architecture:** Shared package `pkg/cmd/selfupdate` holds the compat picker, editor-binary resolution, GitHub download/checksum helpers, and both cobra commands. Root passes the embedded CLI `version` string in. Commands must not call `LoadConfigAndContext`. `install.sh` is not modified.

**Tech Stack:** Go 1.26, cobra, huh, `golang.org/x/mod/semver`, `github.com/mattn/go-isatty` (already in go.mod as indirect — add as a direct require if used).

**Spec:** `docs/superpowers/specs/2026-08-26-cli-update-and-vscode-extension-design.md`

## Global Constraints

- Do **not** change `install.sh` (no vsix, no prompt).
- Do **not** follow red-green TDD. Implement, then add the tests named in each task. Still write real behaviour tests for the picker and the non-TTY `--yes` gate.
- Do **not** read or write `~/.versori/**`, project `.versori`, or editor user settings. Do not call `LoadConfigAndContext`.
- No user-facing `--force` on `versori vscode install`. Editor CLI still gets `--install-extension <vsix> --force`.
- Confirm default **No**. `--yes` / `--confirm` skip the prompt only (`pkg/cmd/flags.AddSkipPromptFlags`).
- Compat URL: `https://raw.githubusercontent.com/versori/cli/main/vscode-compat.json`
- Curl fallback copy: `curl -fsSL https://raw.githubusercontent.com/versori/cli/main/install.sh | sh`
- Extension id: `versori.versori-vscode`
- Extension GitHub repo: `versori/versori-vscode-extension`; CLI repo: `versori/cli`
- Checksum required for CLI tarball and vsix.
- `versori update` picker uses the **target** CLI version, not in-memory `version` after overwrite.
- Binary self-update: darwin/linux only; `sudo` moves onto the already-resolved path (never `sudo versori update`).
- Fail-soft vsix on `update`; fail-hard on `vscode install`.
- Commits use `feat:` / `fix:` on title and body.
- No live GitHub and no live `code`/`cursor` in unit tests — inject HTTP clients / fake binaries.

## File structure

- Create: `vscode-compat.json` (repo root)
- Create: `pkg/cmd/selfupdate/compat.go` — table types, fetch, `PickVSIX`
- Create: `pkg/cmd/selfupdate/compat_test.go`
- Create: `pkg/cmd/selfupdate/editors.go` — resolve `code` / `cursor`
- Create: `pkg/cmd/selfupdate/editors_test.go`
- Create: `pkg/cmd/selfupdate/github.go` — latest tag, download+SHA-256, tarball extract, vsix
- Create: `pkg/cmd/selfupdate/github_test.go` — `httptest` only
- Create: `pkg/cmd/selfupdate/install_vsix.go` — list-extensions + install-extension
- Create: `pkg/cmd/selfupdate/install_vsix_test.go`
- Create: `pkg/cmd/selfupdate/update.go` — `versori update`
- Create: `pkg/cmd/selfupdate/vscode.go` — `versori vscode` + `install`
- Create: `pkg/cmd/selfupdate/prompt.go` — huh / non-TTY
- Modify: `pkg/cmd/root.go` — register both commands, pass `version`
- Modify: `README.md` — update + vscode install
- Modify: `../user-docs/latest/cli/installation.mdx` if that sibling repo exists
- Run: `make generate` (do not hand-edit `docs/*.md`)
- Do not modify: `install.sh`, `pkg/cmd/config/config.go`

---

### Task 1: Compat table and picker

**Files:**
- Create: `vscode-compat.json`
- Create: `pkg/cmd/selfupdate/compat.go`
- Create: `pkg/cmd/selfupdate/compat_test.go`

**Interfaces:**
- Consumes: nothing
- Produces:
  - `type CompatTable struct { ExtensionID string \`json:"extension_id"\`; Releases []CompatRelease \`json:"releases"\` }`
  - `type CompatRelease struct { Vsix string \`json:"vsix"\`; MinCLI string \`json:"min_cli"\` }`
  - `func ParseCompat(data []byte) (CompatTable, error)`
  - `func PickVSIX(cliVersion string, table CompatTable) (vsixTag string, ok bool)` — newest vsix whose `min_cli <= cliVersion` (semver, optional leading `v`)
  - `const DefaultCompatURL = "https://raw.githubusercontent.com/versori/cli/main/vscode-compat.json"`
  - `func TooOldMessage() string` containing `versori update` and `curl -fsSL https://raw.githubusercontent.com/versori/cli/main/install.sh | sh`
  - `func FetchCompat(client *http.Client, url string) (CompatTable, error)` — fail closed on non-200 or parse error

- [ ] **Step 1: Implement picker + JSON + fetch**

`vscode-compat.json`:

```json
{
  "extension_id": "versori.versori-vscode",
  "releases": [
    { "vsix": "0.1.0", "min_cli": "0.1.0" }
  ]
}
```

Canon versions with `golang.org/x/mod/semver` (add `v` if missing). Invalid versions are skipped, not a panic. Empty table → `ok false`.

`FetchCompat` uses the passed client (nil → `http.DefaultClient`).

- [ ] **Step 2: Tests (after implementation)**

Cover spec §3 table plus empty table, equal min_cli, and newer vsix with higher min_cli left unpicked:

```go
// CLI 0.0.9 vs vsix 0.1.0 min 0.1.0 → ok false
// CLI 0.1.0 → vsix 0.1.0
// CLI 0.1.1 → vsix 0.1.0
// CLI 0.1.0 with extra vsix 0.4.0 min 0.3.0 → vsix 0.1.0
// CLI 0.3.0 with those two rows → vsix 0.4.0
```

`ParseCompat` rejects malformed JSON. `FetchCompat` against `httptest` 404 returns error.

Run: `go test ./pkg/cmd/selfupdate/ -count=1`

- [ ] **Step 3: Commit**

```bash
git add vscode-compat.json pkg/cmd/selfupdate/compat.go pkg/cmd/selfupdate/compat_test.go go.mod go.sum
git commit -m "$(cat <<'EOF'
feat: pick a vsix from the CLI compatibility table

feat: newest vsix whose min_cli still fits the running or target CLI
EOF
)"
```

---

### Task 2: Editor resolution and GitHub downloads

**Files:**
- Create: `pkg/cmd/selfupdate/editors.go`
- Create: `pkg/cmd/selfupdate/editors_test.go`
- Create: `pkg/cmd/selfupdate/github.go`
- Create: `pkg/cmd/selfupdate/github_test.go`

**Interfaces:**
- Consumes: Task 1 checksum helper not required here
- Produces:
  - `type EditorKind string` with `EditorVSCode` / `EditorCursor` (`"VS Code"` / `"Cursor"` display names)
  - `type Editor struct { Kind EditorKind; Path string }`
  - `func ResolveEditors(vscodePath, cursorPath string) ([]Editor, error)` — flag set + missing/not-executable → error (caller maps install vs update). Search PATH then well-known paths from spec §5. Skip missing well-known paths.
  - `func LookEditor(kind EditorKind, flagPath string) (path string, flagged bool, err error)` if splitting is cleaner
  - `type GitHub struct { Client *http.Client; CLIRepo string; ExtRepo string }` defaults `versori/cli` and `versori/versori-vscode-extension`
  - `func (g *GitHub) LatestTag(repo string) (string, error)` — GET `https://api.github.com/repos/{repo}/releases/latest`, JSON `tag_name`
  - `func (g *GitHub) DownloadReleaseFile(repo, tag, name, dest string) error`
  - `func VerifySHA256(path, checksumsPath, filename string) error` — grep filename in checksums.txt, compare
  - `func CLITarballName(version, goos, goarch string) string` — `cli_{stripped}_{os}_{arch}.tar.gz` matching `install.sh` (`amd64`/`arm64`, `linux`/`darwin`)
  - `func ExtractVersoriBinary(tarball, destDir string) (binaryPath string, err error)`
  - `func PickVSIXAsset(names []string) (string, error)` — prefer `versori-vscode-*.vsix`, else unique `.vsix`

Well-known paths (verbatim from spec):

| Editor | Darwin | Linux | Windows |
|---|---|---|---|
| VS Code | `/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code` then `/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code` | `/usr/bin/code`, `/usr/share/code/bin/code` | `%LOCALAPPDATA%\Programs\Microsoft VS Code\bin\code.cmd` |
| Cursor | `/Applications/Cursor.app/Contents/Resources/app/bin/cursor` | `/usr/bin/cursor`, `/usr/share/cursor/bin/cursor` | `%LOCALAPPDATA%\Programs\cursor\resources\app\bin\cursor.cmd` |

- [ ] **Step 1: Implement editors + github helpers**

`exec.LookPath` for PATH. `os.Stat` + executable bit (Windows: file exists). Flags: if non-empty, only that path; must be executable.

GitHub: unauthenticated `http.Get` like `pkg/cmd/skills/download.go`. Latest release JSON decode `tag_name`. Download to dest. SHA-256 via `crypto/sha256` + `encoding/hex`; checksums.txt first field is the hash (same as install.sh `awk '{print $1}'`).

- [ ] **Step 2: Tests**

Editors: temp dir with a fake `code` executable on a custom PATH (`t.Setenv("PATH", ...)`), assert resolve; flag pointing at missing path returns error; flag pointing at the fake binary wins over PATH.

GitHub: `httptest.Server` for latest JSON, file download, checksum match and mismatch. `CLITarballName("v0.0.9", "darwin", "arm64") == "cli_0.0.9_darwin_arm64.tar.gz"`. `PickVSIXAsset`.

Run: `go test ./pkg/cmd/selfupdate/ -count=1`

- [ ] **Step 3: Commit**

```bash
git add pkg/cmd/selfupdate/editors.go pkg/cmd/selfupdate/editors_test.go pkg/cmd/selfupdate/github.go pkg/cmd/selfupdate/github_test.go
git commit -m "$(cat <<'EOF'
feat: resolve editor CLIs and download checksummed GitHub assets

feat: PATH plus well-known code/cursor locations; same tarball names as install.sh
EOF
)"
```

---

### Task 3: `versori vscode install`

**Files:**
- Create: `pkg/cmd/selfupdate/prompt.go`
- Create: `pkg/cmd/selfupdate/install_vsix.go`
- Create: `pkg/cmd/selfupdate/install_vsix_test.go`
- Create: `pkg/cmd/selfupdate/vscode.go`
- Create: `pkg/cmd/selfupdate/vscode_test.go`
- Modify: `pkg/cmd/root.go`

**Interfaces:**
- Consumes: `PickVSIX`, `FetchCompat`, `ResolveEditors`, `GitHub`, `TooOldMessage`
- Produces:
  - `func NewVscodeCommand(cliVersion string) *cobra.Command` — parent `vscode`, child `install`
  - `func chooseEditors(resolved []Editor, yes bool, stdinTTY bool) ([]Editor, error)`
  - `func InstallVSIX(bin, vsixPath string, run func(name string, args ...string) ([]byte, error)) error` — `--install-extension vsix --force`
  - `func HasExtension(bin, id string, run ...) (bool, error)` — `--list-extensions`, contains `versori.versori-vscode`
  - Flags on `install`: `--vscode-path`, `--cursor-path`, `AddSkipPromptFlags`
  - **No** `--force` flag
  - Do not `LoadConfigAndContext`

Prompt copy:

- One VS Code: `Install the Versori extension into VS Code?` default No (`elements.NewConfirm`, value starts `false`)
- One Cursor: `Install the Versori extension into Cursor?`
- Both: `huh` / `elements.NewListSelect` options labels `VS Code`, `Cursor`, `Both`, `Cancel` (Cancel → error, no install)
- None: message that neither VS Code nor Cursor CLI was found; mention `--vscode-path` / `--cursor-path`; `NewExitError`
- Non-TTY (`!isatty.IsTerminal(os.Stdin.Fd())` or golang.org/x/term): if `!yes`, message that a TTY or `--yes` is required; exit non-zero; do not call huh or the editor
- `--yes`: install every resolved editor; no huh

Install flow: fetch compat (fail hard) → `PickVSIX(cliVersion, table)` (fail hard + `TooOldMessage`) → download vsix + checksums from extension repo tag (try `v`+tag then bare tag if 404) → install into chosen editors with `--force` → temp cleanup.

- [ ] **Step 1: Implement command + wire root**

In `GetRootCommand` after `versionCmd`:

```go
rootCmd.AddCommand(selfupdate.NewVscodeCommand(version))
```

No PersistentPreRun that loads config.

- [ ] **Step 2: Tests**

- `chooseEditors` / install cobra: non-TTY without `--yes` returns error and does not invoke a stub runner (inject `run` func).
- `--yes` with one fake editor on PATH calls install-extension with `--force`.
- `HasExtension` true/false from stub stdout.
- Command does not create `$HOME/.versori/config.yaml` (set `t.TempDir()` as HOME).

Run: `go test ./pkg/cmd/selfupdate/ ./pkg/cmd/ -count=1`

- [ ] **Step 3: Commit**

```bash
git add pkg/cmd/selfupdate/prompt.go pkg/cmd/selfupdate/install_vsix.go pkg/cmd/selfupdate/install_vsix_test.go pkg/cmd/selfupdate/vscode.go pkg/cmd/selfupdate/vscode_test.go pkg/cmd/root.go
git commit -m "$(cat <<'EOF'
feat: add versori vscode install

feat: always install the latest compatible vsix into chosen editors
EOF
)"
```

---

### Task 4: `versori update` plus docs

**Files:**
- Create: `pkg/cmd/selfupdate/update.go`
- Create: `pkg/cmd/selfupdate/update_test.go`
- Modify: `pkg/cmd/root.go`
- Modify: `README.md`
- Modify: `../user-docs/latest/cli/installation.mdx` when that path exists (sibling repo; do not `git add` it in this repo)
- Generated: run `make generate` and commit `docs/` output

**Interfaces:**
- Consumes: all prior helpers
- Produces: `func NewUpdateCommand(cliVersion string) *cobra.Command`
  - Flags: `--version` (CLI tag pin), `--vscode-path`, `--cursor-path`
  - No extension prompt
  - Windows (`runtime.GOOS == "windows"`): print that Windows should download a release zip; skip binary replace; still run vsix pass
  - Resolve executable: `os.Executable` + `filepath.EvalSymlinks`
  - Target version: flag or `LatestTag("versori/cli")`
  - If already that version (compare `cliVersion` to target after stripping `v`): skip download, still vsix pass. For “already current”, use `cliVersion` passed into the command (the built binary’s version) compared to target — do **not** re-read argv. After a replace in the same process, vsix pass must use **target**, not `cliVersion`.
  - Download tarball + checksums, verify, extract, chmod, rename over dest. If `os.Rename` / write fails with permission: `exec.Command("sudo", "mv", tmpBinary, dest)` (resolved dest). Failure → exit with curl one-liner. No half-written dest (write to dest+`.new` then rename).
  - Vsix pass: `ResolveEditors` — flag errors **warn** and skip that editor. No editors → one line, success. For each editor `HasExtension` → if yes, `PickVSIX(targetVersion, table)`, download, `InstallVSIX`. Failures warn, command still 0 if binary step ok.
  - Compat fetch fail: warn, skip vsix

Test that `update` would pass **target** into `PickVSIX` (unit-test a small func `vsixCLIVersion(current, target string) string { return target }` or the install-vsix-for-update helper with a stub picker).

- [ ] **Step 1: Implement `versori update` and register it**

```go
rootCmd.AddCommand(selfupdate.NewUpdateCommand(version))
```

- [ ] **Step 2: Tests**

Target-version vs process-version: helper or stubbed update vsix step records the CLI version passed to PickVSIX equals `--version` / target, not a fake “old” current. Non-writable dest is hard — skip live sudo; test `CLITarballName` already in Task 2.

HOME temp: update command construction does not write config.yaml.

Run: `go test ./pkg/cmd/selfupdate/ ./pkg/cmd/ -count=1`

- [ ] **Step 3: Docs**

README after the curl pin example, add:

```markdown
### Update

Once installed, update the CLI in place:

```sh
versori update
```

Pin a version with `versori update --version v0.0.9`. This overwrites the `versori` binary you ran (not always `/usr/local/bin`). It also refreshes the Versori VS Code/Cursor extension if it is already installed.

Install the editor extension (prompts; default No):

```sh
versori vscode install
```

Pass `--vscode-path` / `--cursor-path` if `code` / `cursor` are not on your PATH. `--yes` skips the prompt (non-interactive).
```

Same idea in `user-docs/latest/cli/installation.mdx` if present.

```sh
make generate
```

Do not edit `docs/*.md` by hand except what generate writes. Do not change `install.sh`.

- [ ] **Step 4: Commit**

```bash
git add pkg/cmd/selfupdate/update.go pkg/cmd/selfupdate/update_test.go pkg/cmd/root.go README.md docs/
git commit -m "$(cat <<'EOF'
feat: add versori update and document editor install

feat: replace this binary then refresh an already-installed vsix
EOF
)"
```

If user-docs was edited, commit **in that repo** separately (do not add it to the CLI index).

---

## Self-review vs spec

| Spec | Task |
|---|---|
| §1 command split, install.sh untouched | 3, 4, Global Constraints |
| §2 config isolation | 3, 4 tests + no LoadConfigAndContext |
| §3 compat table + picker examples | 1 |
| §4 update binary + vsix-if-installed + target version | 4 |
| §5 vscode install prompts, flags, vsix download | 2, 3 |
| §6 error matrix | 3, 4 |
| §7 docs | 4 |
| §8 tests | 1–4 |
