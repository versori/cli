# Versori CLI Reference

## CLI Commands

### `versori status`
Check if the Versori CLI is installed. Returns binary path and help output. Run first to verify CLI availability.

### `versori context select <context>`
Switch context. Uses `VERSORI_DEFAULT_CONTEXT` env var if set.

### `versori projects list`
List all projects in the current context. Use this to discover project IDs when the user doesn't know them.

### `versori projects create --name <name>`
Create a new project. Returns a 26-character ULID project ID.

### `versori project sync --directory <dir> --project <id>`
Download project files from the Versori platform to a local directory.

**WARNING: `sync` WILL DELETE any local files in the target directory that are not present in the platform.** Always use `--dry-run` first to preview what will be updated or deleted before executing for real.

Use this when the user wants to pull down an existing project to edit locally. After syncing, the user can edit the code and redeploy.

### `versori projects systems list --project <id> --environment <env>`
List systems linked to a project. Returns the systems configured in the given project and environment.

Run this before generating workflow code to discover available system names. If a required system is missing from the list, **do not generate code for it** — instead, ask the user for the name of their org, then give them the direct link `https://ai.versori.com/integrations/<project-id>?org=<org>` to configure the missing systems before proceeding.

### `versori projects systems bootstrap --file <path> --project <id> [--system-overrides <json>]`
Bootstrap systems in a project from a research context file.

The `--file` flag is required and should point to the research document (typically `versori-research/research.md`). The `--project` flag provides the project ID; it defaults from `.versori` when inside a synced project directory.

The `--system-overrides` flag accepts a JSON string keyed by system name, where each value is an object of configuration overrides for that system. For example: `--system-overrides '{"Shopify": {"base_url": "https://my-store.myshopify.com/admin/api/2024-01"}}'`. This flag is optional — omit it if no systems require user-specific configuration. The value must be valid JSON.

**Before running this command**, review the research document's System & Authentication section. If any system has a base URL or configuration that depends on a user-specific value (shop domain, subdomain, instance URL, tenant ID, etc.), ask the user for those values first. Pass the confirmed values via the `--system-overrides` flag rather than modifying the research document.

**This command creates systems in the project.** Always confirm with the user before running. After bootstrapping, always run `versori projects systems list` to verify the created systems before proceeding to code generation.

### `versori connections create`

Create a connection for a system in a project. Run this after `versori projects systems list`, once per system.

**Required flags:**
- `--project <id>` — Project ID
- `--environment <env>` — Environment name (e.g. `production`)
- `--name <name>` — Connection name (must match a system name from `systems list`)
- `--template-id <id>` — Template ID (from `systems list` output)

**Optional flags:**
- `--bypass` — Bypass connection validation (**use this for now** — connections are in active development)
- `--external-id <id>` — External identifier
- `--base-url <url>` — Base URL for the connection
- `--api-key <key>` — API key for authentication
- `--username <user>` / `--password <pass>` — Basic auth credentials

**Always use `--bypass`** while connections are in active development. The `--template-id` value comes from the `systems list` output.

### `versori projects versions list --project <id>`
List the most recent deployed versions of a project.

Run this before deploying to see what versions already exist so you can pick the next version number. Version numbers are plain integers (e.g. `1`, `2`, `3`).

### `versori projects assets list --project <id>`
List all assets for a project. Use to discover what assets are already uploaded before uploading or downloading.

### `versori projects assets upload --file <path> --project <id>`
Upload a file as a project asset. Assets can be used as context by Versori AI agents. The `--file` flag is required. `--project` defaults from `.versori` when inside a synced project directory. Use this after research to upload the research document as a project asset.

### `versori projects assets download --asset <name> --directory <dir> --project <id>`
Download an asset by name. The `--asset` flag is required. `--directory` defaults to `versori-research`. `--project` defaults from `.versori` when inside a synced project directory. Use when pulling down research or other context files from an existing project.

### `versori project deploy -d <dir> --project=<id> --environment <env> --version <ver>`
Deploy a project.

**Defaults:**
- Directory: `.`
- Environment: `production` (or `VERSORI_DEFAULT_ENVIRONMENT`)
- Version: `1`

Add `--dry-run` to show what would happen without executing.

## Recommended `.gitignore`

Both `sync` and `deploy` respect `.gitignore` — files matched by it will not be deployed or deleted during a sync. Always ensure a `.gitignore` exists in the project directory. Minimum recommended content:

```
node_modules/
dist/
.env
production.env
.env.local
.git/
.vscode/
.cursor/
```

## Environment Variables

| Variable | Purpose |
|---|---|
| `VERSORI_DEFAULT_CONTEXT` | Default context for selection |
| `VERSORI_DEFAULT_ENVIRONMENT` | Default deploy environment (fallback: `production`) |
| `VERSORI_CLI_PATH` | Custom path to the versori binary |

## Deployment Safety

**ALWAYS confirm before deploying** unless the user explicitly says "deploy", "ship it", or "go ahead".

Example confirmation: _"I've prepared the deployment command. Would you like me to deploy to production?"_

Consider using `--dry-run` first when intent is ambiguous.

## Workflow

**New project:**
```bash
# 1. (Optional) Check CLI availability
versori status

# 2. Select context if needed
versori context select demo

# 3. Create project → returns projectId
versori projects create --name "my-integration"
# → Project ID: 01KH6HD9QNAT57MGEPYG4CY9J5

# 4. After research phase produces versori-research/research.md,
#    review System & Authentication for user-specific config (shop domains, subdomains, etc.)
#    Ask the user for any required values before bootstrapping.
#    Then bootstrap systems from the research document (confirm with user first)
#    Pass user-specific values via --system-overrides (omit if none needed)
versori projects systems bootstrap --file versori-research/research.md --project 01KH6HD9QNAT57MGEPYG4CY9J5 \
  --system-overrides '{"Shopify": {"base_url": "https://my-store.myshopify.com/admin/api/2024-01"}}'

# 5. List systems to verify what was bootstrapped (note the template-id for each)
versori projects systems list --project 01KH6HD9QNAT57MGEPYG4CY9J5 --environment production
# → shopify (template-id: abc123), postgres (template-id: def456)

# 5.5. Upload the research document as a project asset
versori projects assets upload --file versori-research/research.md --project 01KH6HD9QNAT57MGEPYG4CY9J5

# 6. Create connections for each system (use --bypass while connections are in active development)
versori connections create --project 01KH6HD9QNAT57MGEPYG4CY9J5 --environment production --name shopify --template-id abc123 --bypass
versori connections create --project 01KH6HD9QNAT57MGEPYG4CY9J5 --environment production --name postgres --template-id def456 --bypass

# 7. List existing versions to determine the next version number
versori projects versions list --project 01KH6HD9QNAT57MGEPYG4CY9J5
# → (no versions yet — use version 1)

# 8. After writing code and confirming with user, deploy
versori project deploy -d . --project=01KH6HD9QNAT57MGEPYG4CY9J5 --environment production --version 1
```

**Existing project (pull down and edit):**
```bash
# 1. Find the project ID
versori projects list
# → 01KH6HD9QNAT57MGEPYG4CY9J5  shopify-sync

# 2. Dry-run sync to preview changes
versori project sync --directory shopify-sync/01KH6HD9QNAT57MGEPYG4CY9J5 --project 01KH6HD9QNAT57MGEPYG4CY9J5 --dry-run

# 3. On confirmation, sync for real
versori project sync --directory shopify-sync/01KH6HD9QNAT57MGEPYG4CY9J5 --project 01KH6HD9QNAT57MGEPYG4CY9J5

# 4. Download existing research/context assets
versori projects assets download --asset research.md --project 01KH6HD9QNAT57MGEPYG4CY9J5

# 5. List systems, create connections (with --bypass) if needed, edit code, list versions to pick next version number, then deploy
versori projects versions list --project 01KH6HD9QNAT57MGEPYG4CY9J5
# → version 3, version 2, version 1 → use version 4
```

## Example Interactions

**Create and deploy:**
```
User: "Create a project called 'shopify-sync' and deploy the code I wrote"
1. Run: versori projects create --name "shopify-sync"
2. Note the returned project ID
3. Ask: "Created project 01KH6HD9QNAT57MGEPYG4CY9J5. Deploy to production now?"
4. On confirmation: versori project deploy -d . --project=01KH6HD9QNAT57MGEPYG4CY9J5 --environment production --version 1
```

**Dry-run:**
```
User: "What would the deployment command look like?"
→ versori project deploy -d . --project=01KH6HD9QNAT57MGEPYG4CY9J5 --environment production --version 1 --dry-run
```

**List projects to find an ID:**
```
User: "I want to edit my shopify-sync project but I don't know the ID"
1. Run: versori projects list
   → 01KH6HD9QNAT57MGEPYG4CY9J5  shopify-sync
2. "Found it — project ID is 01KH6HD9QNAT57MGEPYG4CY9J5. Want me to pull it down locally?"
```

**Sync an existing project:**
```
User: "Pull down project 01KH6HD... to edit locally"
1. Run: versori project sync --directory shopify-sync/01KH6HD... --project 01KH6HD... --dry-run
   → shows files that will be updated/deleted
2. "Here's what sync will change. Shall I go ahead?"
3. On confirmation: versori project sync --directory shopify-sync/01KH6HD... --project 01KH6HD...
```

**Check versions before deploying:**
```
User: "Deploy my updated code to project 01KH6HD..."
1. Run: versori projects versions list --project 01KH6HD...
   → version 2, version 1
2. "Latest version is 2. Deploy as version 3 to production?"
3. On confirmation: versori project deploy -d . --project=01KH6HD... --environment production --version 3
```

**Bootstrap systems from research:**
```
User: "I've finished the research for project 01KH6HD..., set up the systems"
1. Review the System & Authentication section of the research document
2. If any system needs user-specific config, ask first:
   → "The research doc lists Shopify, which needs your shop domain (e.g. my-store.myshopify.com). What's your Shopify shop domain?"
3. Build the overrides JSON from confirmed values:
   `{"Shopify": {"base_url": "https://my-store.myshopify.com/admin/api/2024-01"}}`
4. Run: versori projects systems bootstrap --file versori-research/research.md --project 01KH6HD... --system-overrides '{"Shopify": {"base_url": "https://my-store.myshopify.com/admin/api/2024-01"}}'
5. Run: versori projects systems list --project 01KH6HD... --environment production
6. "Bootstrapped 2 systems: 'shopify' and 'snowflake'. Ready to generate workflow code?"
```

**Upload research after completing it:**
```
User: "I've finished the research, upload it to the project"
1. Run: versori projects assets upload --file versori-research/research.md --project 01KH6HD...
2. "Uploaded research.md as a project asset."
```

**Download research from an existing project:**
```
User: "Pull down the research for project 01KH6HD..."
1. Run: versori projects assets list --project 01KH6HD...
   → research.md
2. Run: versori projects assets download --asset research.md --project 01KH6HD...
3. "Downloaded research.md to versori-research/. Ready to review."
```

**Create connections after bootstrap:**
```
User: "Systems are bootstrapped for project 01KH6HD..., now set up connections"
1. Run: versori projects systems list --project 01KH6HD... --environment production
   → shopify (template-id: abc123), postgres (template-id: def456)
2. Run: versori connections create --project 01KH6HD... --environment production --name shopify --template-id abc123 --bypass
3. Run: versori connections create --project 01KH6HD... --environment production --name postgres --template-id def456 --bypass
4. "Created connections for shopify and postgres (with --bypass). Ready to generate workflow code?"
```

**Check systems before writing code:**
```
User: "Write a workflow to sync Shopify orders to Snowflake for project 01KH6HD..."
1. Run: versori projects systems list --project 01KH6HD... --environment production
   → shopify
2. "snowflake" is missing from the project systems.
   → "Your project has a 'shopify' system, but no 'snowflake' system is configured.
      What's the name of your org? I'll give you a direct link to add it."
3. User replies: "jay-benchmarking"
   → "Add it here: https://ai.versori.com/integrations/01KH6HD...?org=jay-benchmarking
      Then come back and I'll generate the workflow code."
```
