# Versori CLI Reference

## Index

- **Project lifecycle**: `projects list/create`, `project sync/deploy`, `projects versions list`
- **Systems & auth**: `systems create`, `systems add-auth-scheme`, `projects systems bootstrap/list/add/update-connection-template/list-connections/connect/delete-connection-template`
- **Connections**: `connections create/list`
- **End-users & activations**: `users create/list`, `projects users activate/deactivate/list/details/set-variable` (aliased under `projects activations`)
- **Dynamic-variable schema**: `projects variables list/add/update/remove/get/set`
- **Assets**: `projects assets list/upload/download`
- **Reference material**: `.gitignore`, environment variables, deployment safety, the `.versori` file, workflow recipes, example interactions

## CLI Commands

### `versori context select <context>`

Switch context. Uses `VERSORI_DEFAULT_CONTEXT` env var if set. If the user asks to switch contexts you can use this command to switch to a different context.

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

### `versori systems create [--name <name>] [--domain <domain>] [--template-base-url <url>]`

Create an **org-scoped system** from scratch (the manual alternative to `bootstrap`, which infers and creates systems from a research markdown file). All flags are optional; the command interactively prompts for any missing value. Use this when adding a single system without the research-document flow.

After creating, attach an auth scheme with `versori systems add-auth-scheme` (skip if the system is `none`), then bind it into a project with `versori projects systems add`.

### `versori systems add-auth-scheme --system-id <id> --type <type> --name <name> [type-specific flags]`

Add (or update) an auth-scheme config on an existing system. Use this when bootstrap produced a system with `authSchemeConfigs: []` (verify with `versori projects systems list -o yaml`).

Supported `--type` values: `none`, `api-key`, `basic-auth`, `oauth2`, `hmac`, `certificate`. Type-specific flags follow `--<type>.<field>` naming, e.g. `--api-key.in header --api-key.name X-API-Key` for an API-key scheme.

### `versori projects systems add --project <id> --system <system-id> --name <ct-name> --environment <env> [--dynamic]`

Create a **connection template** (a.k.a. EnvironmentSystem) by linking an existing org system to a project environment. Manual alternative to `bootstrap` for binding a single system. Pass `--dynamic` if the system runs against per-end-user embedded connections (no static connection required at deploy time).

Full manual lifecycle for adding one system end-to-end:

1. `versori systems create` — create the org system
2. `versori systems add-auth-scheme --system <id> ...` — attach auth (skip if `none`)
3. `versori projects systems add --project <id> --system <id> --name <ct-name> --environment <env>` — bind into a project as a connection template
4. `versori connections create --project <id> --environment <env> --template-id <ct-id> ...` — fill the template with a credential

### `versori connections create`

Create a connection for a system in a project. Run this after `versori projects systems list`, once per system.

**Required flags:**

- `--project <id>` — Project ID
- `--environment <env>` — Environment name (e.g. `production`)
- `--name <name>` — Connection name. Just a human-readable reference shown in the UI; doesn't impact functionality. To avoid name conflicts, suffix it with random characters.
- `--template-id <id>` — Template ID (from `systems list` output)

**Optional flags:**

- `--bypass` — Create a no-auth connection. **Mutually exclusive with credential flags** (`--api-key`, `--username`/`--password`, `--client-id`/`--client-secret`). When passed, all credential flags are silently ignored and the connection is created with no authentication. Only use when the auth scheme type is `none` or the user explicitly wants to skip credentials.
- `--external-id <id>` — End-user external ID. When provided, the connection is created as an **embedded (dynamic) per-end-user connection** bound to that end-user. When omitted, the connection is created as a **static** project-wide connection used by every activation. Static is the default; pass `--external-id` only when building multi-tenant projects where each end-user brings their own credentials.
- `--base-url <url>` — Base URL for the connection
- `--api-key <key>` — API key for authentication
- `--username <user>` / `--password <pass>` — Basic auth credentials
- `--client-id <id>` / `--client-secret <secret>` — OAuth2 client credentials
- `--token-url <url>` — OAuth2 token URL
- `--env-file <path>` — Path to `.env` file for resolving `$VARIABLE` references in credential flags (default: `.env`)

**OAuth2 `authorization_code` grants cannot be completed from the CLI.** If
the auth scheme in `versori projects systems list -o yaml` shows
`grant.type: authorizationCode`, do not run `versori connections create` for
that system — the command will hang waiting on a redirect the CLI can't
drive. Instead, direct the user to the project in the Versori UI
(`https://ai.versori.com/integrations/<project-id>?org=<org>`) and have them
authorize the system there. Client-credentials (`clientCredentials`) grants
are fine to create from the CLI.

**Variable references:** Credential flags accept `$VARIABLE` or `${VARIABLE}` syntax. The CLI resolves these from the `.env` file (or the process environment as fallback) at runtime, so the actual secret never appears in the command.

**Auth-scheme-to-env-var mapping:** When generating `.env.example` files from `systems list` output, use the system's `AuthSchemeConfigs.Type` to determine which variables are needed. Uppercase the system name and replace hyphens with underscores for the variable prefix.

| `AuthSchemeConfigs.Type` | Required env vars | `connections create` flags |
|---|---|---|
| `api-key` | `<SYSTEM>_API_KEY` | `--api-key '$<SYSTEM>_API_KEY'` |
| `basic-auth` | `<SYSTEM>_USERNAME`, `<SYSTEM>_PASSWORD` | `--username '$<SYSTEM>_USERNAME' --password '$<SYSTEM>_PASSWORD'` |
| `oauth2` (`authorization_code` grant) | _(none — use UI)_ | **Do not run `versori connections create`**; create the connection from the Versori UI instead. The CLI cannot complete the browser redirect and the command will hang. |
| `oauth2` (`client_credentials` grant) | `<SYSTEM>_CLIENT_ID`, `<SYSTEM>_CLIENT_SECRET` | `--client-id '$<SYSTEM>_CLIENT_ID' --client-secret '$<SYSTEM>_CLIENT_SECRET'` |
| `none` | _(none)_ | `--bypass` (only valid for `none` auth type — do not combine with credential flags) |

For `oauth2` with the `client_credentials` grant, read the token URL from the `versori projects systems list -o yaml` output and pass it via `--token-url`. For `oauth2` with the `authorization_code` grant, stop and direct the user to the Versori UI — see the OAuth2 callout above.

```bash
# Example: store secrets in .env, reference them in the command
# .env file contains: SHOPIFY_API_KEY=sk-12345...
versori connections create --project <id> --environment production \
  --name shopify --template-id <tid> --api-key '$SHOPIFY_API_KEY'
```

**Default to real credentials.** Generate a `.env.example`, have the user fill in `.env`, and use variable references. Offer `--bypass` as a fallback if the user prefers to skip credentials. The `--template-id` value comes from the `versori projects systems list` output.

### `versori connections list [--system <id>] [--end-user <external-id>]`

List connections in the current organisation context. Both filters are optional:

- `--system <id>` — show only connections on one system
- `--end-user <external-id>` — show only embedded connections owned by one end-user (the default unfiltered output lists static project-wide connections only; embedded connections only appear when this filter is provided)

### `versori projects systems update-connection-template --project <id> --template <template-id> [--name <name>] [--dynamic] [--auth-scheme-config-id <id>]`

Update a connection template (a.k.a. EnvironmentSystem) on a project. Toggles the `dynamic` flag, renames, or swaps the linked auth-scheme-config. Pass at least one update flag.

The `dynamic` flag controls **deploy-config build behaviour**:

- `dynamic: false` (default): the deploy-config builder requires a static connection on this system. Deploys fail with `Failed to build config json: missing connections for the following systems: <name>` if none exists.
- `dynamic: true`: deploys don't require a static connection; the system runs against per-end-user embedded connections supplied by activations.

The runtime data plane does **not** check this flag — it's a deploy / UX hint. But because deploys fail without it being right, treat it as a hard gate.

```bash
versori projects systems update-connection-template \
  --project <id> --template <template-id> --dynamic
```

### `versori projects systems list-connections --project <id> --environment <env>`

List **static** connections wired to a project environment, one row per environment system showing the connection's name, ID, base URL, and linked connection-template ID. Embedded (per-end-user) connections do NOT appear here — list those via `versori connections list --end-user <external-id>`.

```bash
versori projects systems list-connections --project 01KH6HD... --environment production
```

### `versori projects systems connect --project <id> --environment <env> --template-id <ct-id> --connection-id <conn-id>`

Bind an existing connection to a connection template on a specific environment. Rarely needed in normal flows — `versori connections create` already binds the connection to the CT it was created against. Use this when re-wiring a CT to a different existing connection (e.g. after rotating credentials in another environment).

```bash
versori projects systems connect --project 01KH6HD... --environment production \
  --template-id 01KTPL... --connection-id 01KCONN...
```

### `versori projects systems delete-connection-template --project <id> --template <ct-id>`

Delete a connection template from a project. The org-scoped system itself is NOT deleted — only the project's binding to it. Aliased as `versori projects systems remove` (same handler, identical behaviour).

```bash
versori projects systems delete-connection-template --project 01KH6HD... --template 01KTPL...
```

### `versori users create -e <external-id> -n <display-name>`

Create an org-scoped end-user. `external-id` is your handle (anything stable per tenant, e.g. `merchant-42` or a UUID); `display-name` is human-readable. Re-creating with the same `external-id` errors. End-users are org-global; the same one can be activated on many projects/environments. Output is empty on success — verify with `versori users list` or by activating.

### `versori users list`

List all end-users in the current organisation.

### `versori projects users list --project <id> --environment <env>`

List **activations** (end-user ↔ environment links) on a project environment. Each row shows the end-user's display name, external ID, the activation ID, and the activation's dynamic variables.

Alias: `versori projects activations list`.

### `versori projects users details --project <id> --environment <env> --external-id <user-external-id>`

Show one activation in full, including the connections wired to each environment system and the dynamic-variable bag.

Alias: `versori projects activations details`.

### `versori projects users activate --project <id> --environment <env> --external-id <user-external-id> --connection <system-template-id>=<connection-id>... [--variable key=value]... [--variables-file <path>]`

Create an activation — links an existing end-user to a project environment with one connection per environment system and an optional bag of dynamic variables.

- Pass one `--connection` per environment system (list them with `versori projects systems list -o yaml`; copy each system's `connectionTemplateId`, then the existing connection's ID from `versori connections list --end-user <user-external-id>`).
- `--variable key=value` is repeatable; values are parsed as JSON when valid (so `42`, `true`, `{"a":1}` all work), else treated as strings. `--variables-file` reads a flat JSON object; both can be combined (`--variable` wins on conflicts).
- Variable keys must already exist in the project's `DynamicVariablesSchema` (manage with `versori projects variables list/add/update/remove`; or `set` for raw JSON-Schema shapes); unknown keys fail server-side validation.

Alias: `versori projects activations create` (with subcommand aliases `activate`, `new`).

```bash
versori projects users activate --project <id> --environment production \
  --external-id merchant-42 \
  --connection 01KTEMPLATE...=01KCONNECTION... \
  --variable tenant_org_id=01KORG... \
  --variable channel_id=aboutyou
```

### `versori projects users set-variable --project <id> --environment <env> --external-id <user-external-id> --name <key> --value <value>`

Set a single dynamic variable on an existing activation. `--value` is parsed as JSON when valid, else treated as a string. Variable updates take effect at runtime immediately — no redeploy.

Alias: `versori projects activations set-variable`.

### `versori projects users deactivate --project <id> --environment <env> --external-id <user-external-id>`

Delete an activation. The end-user and any embedded connections are not deleted — only the link to the environment. Re-activate with `versori projects users activate`.

Alias: `versori projects activations delete` (with subcommand aliases `deactivate`, `rm`).

### `versori projects variables` — DynamicVariablesSchema management

The schema declares which dynamic-variable keys end-user activations on this project are allowed to set (via `set-variable` or the `--variable` flag on `activate`). Two tiers of commands:

- **High-level** (`list` / `add` / `update` / `remove`) — work in terms of Name / Type / Description / Required. **Default to these** in 95% of cases. They never make the user write JSON Schema.
- **Low-level** (`get` / `set`) — operate on the raw JSON Schema. Use only when an advanced JSON-Schema shape is needed (e.g. `enum`, `default`, nested `object`, `patternProperties`) or to bulk-import a hand-crafted schema in CI.

The high-level commands GET-modify-PUT the schema under the hood, so extra JSON-Schema fields previously added via `set` (`enum`, `default`, nested object properties, etc.) are preserved when you `update` a variable without re-specifying its `--type`.

### `versori projects variables list --project <id>`

Print a table of the project's declared variables (Name / Type / Required / Description). Honours the global `-o yaml|json|table` flag.

```bash
versori projects variables list
# Name           Type    Required  Description
# channel_id     string  false     Marketplace channel slug (e.g. "aboutyou")
# tenant_org_id  string  true      Tenant's Versori organisation ID
```

### `versori projects variables add --project <id> [--name <key>] [--type <type>] [--description <text>] [--required] [--items-type <type>] [--field <path>:<type>[:required]]... [--strict]`

Declare a variable. Two modes:

- **Interactive** — omit `--name`; the command prompts for Name / Type / Description / Required and (for object/array types) **recursively** prompts for sub-fields until you submit an empty name. Caps at 5 levels of nesting interactively — use `--field` for deeper.
- **Non-interactive** — pass `--name` and any of the structural flags below.

Top-level types: `string`, `number`, `integer`, `boolean`, `object`, `array`.

**Nested structure** for `--type object` (and `--type array --items-type object`):

- `--field <path>:<type>[:required]` — repeatable. Dotted path is relative to the container's `properties`; missing parent objects are auto-created. Append `:required` to mark the leaf required on its immediate parent. (Interior path segments are never marked required automatically.)
- `--strict` — sets `additionalProperties: false` on every object node in the resulting schema so unknown sub-keys are rejected at activation time. Default is to accept extras.

**Array element type** for `--type array`:

- `--items-type <type>` — element type (any of the top-level types). Omit to accept any element shape.
- When `--items-type object`, `--field` paths describe the item object's properties.

**Existing variables** are updated, not wiped — type/description/required change but any pre-existing nested fields and any advanced JSON-Schema attributes (`enum`, `default`, `format`, `pattern`, `minimum`, etc.) added via `set --file` are preserved on the leaf. Use `remove` + `add` for a clean reset.

```bash
versori projects variables add \
  --name tenant_org_id --type string \
  --description "Tenant's Versori organisation ID" \
  --required
```

Array of strings:

```bash
versori projects variables add --name allowed_emails --type array --items-type string
```

Array of objects with structural validation:

```bash
versori projects variables add --name addresses --type array --items-type object --strict \
  --field street:string:required \
  --field zip:string:required \
  --field country:string
```

Nested object with a sub-object:

```bash
versori projects variables add --name feature_flags --type object --strict \
  --field enabled:boolean:required \
  --field tier:string:required \
  --field metadata.version:string \
  --field metadata.notes:string
```

### `versori projects variables update --project <id> --name <key> [--type <type>] [--description <text>] [--required[=true|false]] [--items-type <type>] [--field <path>:<type>[:required]]... [--strict]`

Update fields on an existing variable. Only flags you explicitly pass are changed — omit a flag to leave that field untouched. Toggle required off with `--required=false`. Clear an existing description with `--description=""`.

**Type changes reset orphans.** Changing `--type` clears any JSON-Schema attributes that don't belong on the new type (so a string variable with `enum` doesn't carry the enum forward after flipping to `object`). The human-meaningful description and the top-level required flag are preserved.

**Field updates are additive.** `--field` declares-or-updates the named sub-field but leaves other sub-fields alone. For a clean reset use `remove` + `add`.

```bash
versori projects variables update --name tenant_org_id --required=false

versori projects variables update --name addresses --field country:string:required

versori projects variables update --name feature_flags --type object --field new_field:string
```

### `versori projects variables remove --project <id> --name <key> [--yes]`

Delete a variable from `properties` and (if present) from `required`. Confirms before deletion unless `--yes` is passed. Aliases: `rm`, `delete`. Activations that previously set the key keep the stored value on their record — only future validation changes.

### `versori projects variables get --project <id>` *(low-level)*

Print the raw `DynamicVariablesSchema` (JSON). Use when you need to inspect advanced JSON-Schema shapes that `list` doesn't surface, or to copy the current schema as a starting point for an edit.

### `versori projects variables set --project <id> --file <schema.json>` *(low-level)*

Replace the entire `DynamicVariablesSchema` (PUT). Anything not in the new schema is removed. Use this only for advanced JSON-Schema shapes or bulk imports — for individual variables prefer `add`. Schema file shape:

```json
{
  "type": "object",
  "properties": {
    "tenant_org_id": { "type": "string" },
    "channel_id":    { "type": "string", "enum": ["aboutyou", "other"] }
  },
  "required": ["tenant_org_id"]
}
```

### `versori projects versions list --project <id>`

List the most recent deployed versions of a project.

Run this before deploying to see what versions already exist so you can pick the next version number. Version numbers are plain integers (e.g. `1`, `2`, `3`).

### `versori projects assets list --project <id>`

List all assets for a project. Use to discover what assets are already uploaded before uploading or downloading.

### `versori projects assets upload --file <path> --project <id>`

Upload a file as a project asset. Assets can be used as context by Versori AI agents. The `--file` flag is required. `--project` defaults from `.versori` when inside a synced project directory. Use this after research to upload the research document as a project asset.

### `versori projects assets download --asset <name> --directory <dir> --project <id>`

Download an asset by name. The `--asset` flag is required. `--directory` defaults to `versori-research`. `--project` defaults from `.versori` when inside a synced project directory. Use when pulling down research or other context files from an existing project.

### `versori project deploy -d <dir> --project=<id> --environment <env> --assets`

Deploy a project and upload the project asset files as well.

**Defaults:**

- Directory: `.`
- Environment: `production` (or `VERSORI_DEFAULT_ENVIRONMENT`)
- Version: Can be left empty and the CLI will generate a name based on the current timestamp.

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
.gitignore
```

`.gitignore` ignores itself on purpose. Without this entry, `versori project sync` deletes the local `.gitignore` (it is not uploaded to the platform), which then exposes `.cursor/` and friends to being deleted on subsequent syncs.

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

## The `.versori` File

When you run `versori project sync`, the CLI creates a `.versori` file in the synced directory. This file contains:

- `project_id` — the project's ULID
- `context` — the CLI context that was active when the project was synced

When a `.versori` file is present in the current directory, the `--project` flag is **optional** for most commands — the CLI reads the project ID automatically. This applies to: `deploy`, `save`, `sync`, `systems list`, `systems bootstrap`, `assets list`, `assets upload`, `assets download`, `logs`, `proxy`, and `versions list`.

**Important:** The `context` stored in `.versori` must match the current CLI context. If you switch contexts, the `.versori` file from a different context will not work — you need to re-sync or switch back.

## Workflow

**Step 1 — Determine the active project:**

```bash
# Check if a .versori file exists in the current directory.
# If it does, the project ID is already known — skip to step 2.

# If no .versori file, ask the user: existing project or new?

# Existing project:
versori projects list
# → 01KH6HD...  shopify-sync
# Let the user pick one, then sync it down (see "Existing project" workflow below)

# New project:
versori projects create --name "my-integration"
# → Project ID: 01KH6HD...
```

**New project (continued):**

```bash
# 1. (Optional) Check CLI availability
versori version

# 2. Research phase — capture API findings in versori-research/research.md
#    (see the "Research Phase" section of SKILL.md for required content)

# 3. After research phase produces versori-research/research.md,
#    review System & Authentication for user-specific config (shop domains, subdomains, etc.)
#    Ask the user for any required values before bootstrapping.
#    Then bootstrap systems from the research document (confirm with user first)
#    Pass user-specific values via --system-overrides (omit if none needed)
versori projects systems bootstrap --file versori-research/research.md --project 01KH6HD... \
  --system-overrides '{"Shopify": {"base_url": "https://my-store.myshopify.com/admin/api/2024-01"}}'

# 4. List systems to verify what was bootstrapped (note the template-id for each)
versori projects systems list --project 01KH6HD... --environment production
# → shopify (template-id: abc123), postgres (template-id: def456)

# 5. Upload the research document as a project asset
versori projects assets upload --file versori-research/research.md --project 01KH6HD...

# 6. Create connections for each system (credentials-first; --bypass available as fallback)
#    Inspect auth types from systems list, generate .env.example, have user fill .env,
#    then create connections with variable references:
versori connections create --project 01KH6HD... --environment production --name shopify --template-id abc123 --api-key '$SHOPIFY_API_KEY'
versori connections create --project 01KH6HD... --environment production --name postgres --template-id def456 --username '$POSTGRES_USERNAME' --password '$POSTGRES_PASSWORD'

# 7. Ensure .gitignore exists (create with recommended content if missing)
#    This prevents node_modules/, dist/, .env from being deployed
#    See "Recommended .gitignore" section above for content

# 8. List existing versions to determine the next version number
versori projects versions list --project 01KH6HD...
# → (no versions yet — use version 1)

# 9. Verify code validity locally
deno install
deno check src/index.ts
# Run tests if applicable
deno test

# 10. After writing code, verifying it, and confirming with user, deploy
versori project deploy -d . --project=01KH6HD... --environment production --version 1
```

**Existing project (pull down and edit):**

```bash
# 1. Find the project ID
versori projects list
# → 01KH6HD...  shopify-sync

# 2. Dry-run sync to preview changes
versori project sync --directory shopify-sync/01KH6HD... --project 01KH6HD... --dry-run

# 3. On confirmation, sync for real file and the projects assets if there are any
versori project sync --directory shopify-sync/01KH6HD... --project 01KH6HD... --assets

# 3.5. Ensure .gitignore exists (create with recommended content if missing)
#      This prevents node_modules/, dist/, .env from being deployed
#      See "Recommended .gitignore" section above for content

# 4. Download existing research/context assets
versori projects assets download --asset research.md --project 01KH6HD...

# 5. List systems, create connections (with real credentials, or --bypass if preferred) if needed, edit code, list versions to pick next version number, then deploy
versori projects versions list --project 01KH6HD...
# → version 3, version 2, version 1 → use version 4

# 6. Verify code validity locally before deploying
deno install
deno check src/index.ts
deno test
```

## Example Interactions

**Create and deploy:**

```
User: "Create a project called 'shopify-sync' and deploy the code I wrote"
1. Run: versori projects create --name "shopify-sync"
2. Create a .gitignore to make sure no user files are deleted
3. Run: versori project sync --project <id> --dry-run /  versori project sync --project <id> to get the .versori file locally
4. Note the returned project ID
5. Run: deno install && deno check src/index.ts (and deno test if applicable) to verify code
6. Ask: "Created project 01KH6HD... and verified code. Deploy to production now?"
7. On confirmation: versori project deploy -d . --project=01KH6HD... --environment production
```

**Dry-run:**

```
User: "What would the deployment command look like?"
→ versori project deploy -d . --project=01KH6HD... --environment production --dry-run
```

**List projects to find an ID:**

```
User: "I want to edit my shopify-sync project but I don't know the ID"
1. Run: versori projects list
   → 01KH6HD...  shopify-sync
2. "Found it — project ID is 01KH6HD.... Want me to pull it down locally?"
```

**Sync an existing project:**

```
User: "Pull down project 01KH6HD... to edit locally"
1. Run: versori project sync --directory shopify-sync/01KH6HD... --project 01KH6HD... --assets --dry-run
   → shows files that will be updated/deleted
2. "Here's what sync will change. Shall I go ahead?"
3. On confirmation: versori project sync --directory shopify-sync/01KH6HD... --project 01KH6HD... --assets
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
   → shopify (template-id: abc123, auth: api-key), postgres (template-id: def456, auth: basic-auth)
2. Generate .env.example with required variables (SHOPIFY_API_KEY, POSTGRES_USERNAME, POSTGRES_PASSWORD)
3. Ask user to copy .env.example to .env and fill in credentials
   → "I've generated .env.example with the required variables. Copy it to .env and fill in your credentials, then let me know when you're ready. Or if you'd prefer to skip credentials for now, I can create bypass connections instead."
4. Once user confirms .env is ready:
   Run: versori connections create --project 01KH6HD... --environment production --name shopify --template-id abc123 --api-key '$SHOPIFY_API_KEY'
   Run: versori connections create --project 01KH6HD... --environment production --name postgres --template-id def456 --username '$POSTGRES_USERNAME' --password '$POSTGRES_PASSWORD'
5. "Created connections for shopify and postgres with real credentials. Ready to generate workflow code?"
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
