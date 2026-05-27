# Versori CLI Reference

## Index

- **Project lifecycle**: `projects list/create/details`, `project sync/deploy`, `projects versions list`
- **Systems & auth**: `systems create`, `systems add-auth-scheme`, `projects systems bootstrap/list/add/update-connection-template/list-connections/connect/delete-connection-template`
- **Connections**: `connections create/list`
- **End-users & activations**: `users create/list`, `projects users activate/deactivate/list/details/set-variable` (aliased under `projects activations`)
- **Dynamic-variable schema**: `projects variables list/add/update/remove/get/set`
- **Assets**: `projects assets list/upload/download`
- **Observability & alerts**: `projects logs`, `notifications channels list/create/delete`, `notifications project list/link/unlink`
- **Reference material**: `.gitignore`, environment variables, deployment safety, the `.versori` file, workflow recipes, example interactions

## CLI Commands

### Agent-safe (non-interactive) invocation

Many commands open an interactive prompt when a flag or positional is omitted. **In non-TTY shells (CI, pipes, agent sandboxes) the prompt blocks on stdin and the command does not return.** Always pass the id/flag explicitly and add `--yes` on any command that also confirms before deleting.

| Command | Pass explicitly (omit → prompt blocks) | Source from |
|---|---|---|
| `versori context add` | `--name`, `--organisation`, `--jwt` | n/a (user-supplied) |
| `versori context select <context-name>` | positional `<context-name>` | `versori context list` |
| `versori context rm <context-name>` | positional `<context-name>` | `versori context list` |
| `versori projects create` | `--name` | n/a (user-supplied) |
| `versori projects details <project-id>` | positional `<project-id>` | `versori projects list -o json` |
| `versori projects deploy` | `--confirm` | n/a |
| `versori projects save` | `--confirm` | n/a |
| `versori projects sync` | `--confirm` | n/a |
| `versori projects versions create` | `--project` (when not in a `.versori` dir) | `versori projects list -o json` |
| `versori projects versions pull` | `--project`, `--version` | `versori projects list -o json` / `versori projects versions list --project <id> -o json` |
| `versori projects variables add` | `--name` (+ `--type` / `--field` for structural shapes) | see the command's full entry below |
| `versori projects users activate` | `--connection` per template + every required `--variable` | see the command's full entry below |
| `versori systems create` | `--name`, `--domain`, `--template-base-url` | n/a (user-supplied) |
| `versori users create` | `--display-name`, `--external-id` | n/a (user-supplied) |
| `versori notifications channels delete` | `--channel-id`, `--yes` | `versori notifications channels list -o json` |
| `versori notifications project link` | `--channel-id`, `--environment` | `versori notifications channels list -o json` |
| `versori notifications project unlink` | `--notification-id`, `--environment`, `--yes` | `versori notifications project list --project <id> -o json` |

Rule of thumb: if `--help` shows an id/flag as optional but the command can't proceed without one, that's an interactive-prompt fallback — pass it explicitly.

### `versori context select <context>`

Switch context. Uses `VERSORI_DEFAULT_CONTEXT` env var if set. If the user asks to switch contexts you can use this command to switch to a different context.

### `versori projects list`

List all projects in the current context. Use this to discover project IDs when the user doesn't know them.

### `versori projects create --name <name>`

Create a new project. Returns a 26-character ULID project ID.

### `versori projects details <project-id>`

Fetch metadata for a single project — name, deployed flag, starred flag, and the list of environments (id, name, public URL, status, provisioner, config, currently deployed version). Returns **metadata only**, not project file contents; to read files use `versori projects files --project <id> -o json` instead.

The id can also be supplied on stdin by passing `-`, e.g. `echo 01KH6HD... | versori projects details -`.

**Agent: always pass the `<project-id>` positional explicitly.** Source the id from `versori projects list -o json` (or the local `.versori` file).

```bash
versori projects details 01KH6HD... -o json
```

### `versori project sync --directory <dir> --project <id>`

Download project files from the Versori platform to a local directory.

**WARNING: `sync` WILL DELETE any local files in the target directory that are not present in the platform.** Always use `--dry-run` first to preview what will be updated or deleted before executing for real.

Use this when the user wants to pull down an existing project to edit locally. After syncing, the user can edit the code and redeploy.

**Cross-project safety.** When `--project` differs from the target directory's `.versori`, `sync` runs an interactive confirmation step before overwriting the directory and re-pinning `.versori`. The step is a human-only safeguard — **agents must always pass `--confirm` on `sync`**; without it the command blocks on stdin in non-interactive contexts. The dir checked is the resolved `--directory` (defaults to cwd), so passing `--directory <path>` shifts the check to `<path>/.versori`.

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

### `versori connections list [--system <id>] [--end-user <ulid-or-external-id>]`

List connections in the current organisation context. Output columns: `ConnectionName / ConnectionId / SystemName / SystemId / BaseUrl` (with `systemName` resolved client-side so you don't have to cross-reference IDs). Both filters are optional:

- `--system <id>` — show only connections on one system
- `--end-user <ulid-or-external-id>` — show only embedded connections owned by one end-user. Accepts either an end-user ULID or your external ID; external IDs are resolved to a ULID client-side before the API call.

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

List **every end-user in the organisation** alongside their activation status on the given project environment. Columns: `DisplayName / ExternalId / Status / ActivationId / UserId / CreatedAt`, where `Status` is `active` if the user has an activation on this environment and `inactive` otherwise. Active rows include the activation ID (and the dynamic-variable bag in JSON/YAML output); inactive rows leave those fields blank. Active users sort to the top, then alphabetically by display name.

Alias: `versori projects activations list`.

### `versori projects users details --project <id> --environment <env> --external-id <user-external-id>`

Show one activation in full, including the connections wired to each environment system and the dynamic-variable bag.

Alias: `versori projects activations details`.

### `versori projects users activate --project <id> --environment <env> --external-id <user-external-id> [--connection <system-template-id>=<connection-id>]... [--variable key=value]... [--variables-file <path>]`

Create an activation — links an existing end-user to a project environment with one connection per environment system and an optional bag of dynamic variables.

- Pass one `--connection` per environment system (list them with `versori projects systems list -o yaml`; copy each system's `connectionTemplateId`, then the existing connection's ID from `versori connections list --end-user <user-external-id>`). **If `--connection` is omitted**, the CLI fetches the environment's connection templates and prompts you to pick a connection per template interactively (auto-selecting when only one candidate exists).
- `--variable key=value` is repeatable; values are parsed as JSON when valid (so `42`, `true`, `{"a":1}` all work), else treated as strings. `--variables-file` reads a flat JSON object; both can be combined (`--variable` wins on conflicts).
- Variable keys must already exist in the project's `DynamicVariablesSchema` (manage with `versori projects variables list/add/update/remove`; or `set` for raw JSON-Schema shapes). The CLI pre-flights validation locally before submitting — unknown keys are rejected up-front in a single message with the exact list, alongside the `versori projects variables list --project <id>` command to inspect the schema. **Missing required variables trigger an interactive prompt** (type-aware: boolean, integer, number, string, or raw JSON for object/array) so the activation isn't sent until they're supplied.

Both interactive prompts run on the same TTY the CLI is invoked from. In non-interactive shells (CI, pipes, agent sandboxes) the prompts will block waiting for input — pass `--connection` and all required `--variable` flags explicitly in those contexts.

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

**Schema must be declared first.** The `--name` key must already exist in the project's `DynamicVariablesSchema` — declare it with `versori projects variables add --project <id> --name <key> --type <type>` before calling `set-variable`. The CLI pre-flights locally and exits with a clear error pointing at `versori projects variables list` / `add` if the key is unknown, so an unknown key never reaches the platform.

Alias: `versori projects activations set-variable`.

### `versori projects users deactivate --project <id> --environment <env> --external-id <user-external-id>`

Delete an activation. The end-user and any embedded connections are not deleted — only the link to the environment. Re-activate with `versori projects users activate`.

Alias: `versori projects activations delete` (with subcommand aliases `deactivate`, `rm`).

### `versori projects variables` — DynamicVariablesSchema management

The schema declares which dynamic-variable keys end-user activations on this project are allowed to set (via `set-variable` or the `--variable` flag on `activate`). Two tiers of commands:

- **High-level** (`list` / `add` / `update` / `remove`) — work in terms of Name / Type / Description / Required. **Default to these** in 95% of cases. They never make the user write JSON Schema.
- **Low-level** (`get` / `set`) — operate on the raw JSON Schema. Use only when an advanced JSON-Schema shape is needed (e.g. `enum`, `default`, nested `object`, `patternProperties`) or to bulk-import a hand-crafted schema in CI.

The high-level commands GET-modify-PUT the schema under the hood, so extra JSON-Schema fields previously added via `set` (`enum`, `default`, nested object properties, etc.) are preserved when you `update` a variable without re-specifying its `--type`.

**Schema-first ordering.** Both `versori projects users activate --variable` and `versori projects users set-variable` pre-flight against this schema and refuse keys that aren't declared. Always run `versori projects variables add` (or `set`) before any activation supplies a value for a new key — there is no "auto-declare on first use".

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

**Cross-project safety.** When `--project` differs from the target directory's `.versori`, `deploy` runs an interactive confirmation step before uploading the local files as a new version of the other project. The step is a human-only safeguard — **agents must always pass `--confirm` on `deploy`**; without it the command blocks on stdin in non-interactive contexts. The dir checked is the resolved `--directory` (defaults to cwd), so passing `--directory <path>` shifts the check to `<path>/.versori`. The same gate (and the same agent rule) applies to `versori projects save`.

### `versori projects logs --environment <env> [--project <id>] [--since <duration>] [--limit <n>] [--search <query>]`

Fetch workflow execution logs for one project + environment. Output is **always one JSON object per line** on stdout, ordered ascending by time — the global `-o` flag is ignored.

**Required:**

- `--environment <env>` — e.g. `production`, `staging`. The CLI rejects the call if omitted (hard error, no picker fallback).

**Optional:**

- `--project <id>` — defaults from `.versori` when inside a synced project directory.
- `--since <duration>` — Go duration window from now (default `24h`). Examples: `30s`, `15m`, `2h30m`.
- `--limit <n>` — cap on returned entries (default `0` = no cap).
- `--search <query>` — server-side filter; useful for narrowing to one execution by ID, a task name, or an error substring.

**There is no `--follow` / `-f` flag.** Logs are pulled once per invocation. To tail while a workflow runs, poll in a loop:

```bash
while true; do
  clear
  versori projects logs --environment production --since 5m --limit 200
  sleep 5
done

# Or with GNU `watch` (on macOS: brew install watch):
watch -n 5 'versori projects logs --environment production --since 5m --limit 200'
```

Each entry has these fields:

| Field | Source | Notes |
|---|---|---|
| `timestamp` | runtime | ISO 8601 in UTC. |
| `severity` | the `ctx.log.<level>` call (or runtime) | One of `debug`, `info`, `warn`, `error`. Crashes outside `ctx.log` are emitted by the runtime as `error` entries. |
| `message` | first arg of `ctx.log.<level>(msg, fields?)` | The static string the workflow code logged. |
| `fields` | second arg + auto-bound | Always includes `executionId`, `activationId`, `externalUserId` (when available); plus anything bound via `ctx.log.child({...})` or passed per-call. |
| `error` | runtime | Populated when an exception bubbled up; usually empty for `ctx.log.*` lines. |

#### Diagnosing a workflow failure from logs

When a user reports a broken workflow, **read logs before reading workflow source.** Logs tell you what actually happened (which task, which input, which upstream response); the source only tells you what the code intends to do.

The pattern is _find the error → pull the full execution trace → walk it top-to-bottom → report the failure point with input + upstream response_.

1. **Pull recent errors and warnings.** Start broad enough to cover the reported time:

   ```bash
   versori projects logs --environment production --since 1h --limit 500 \
     | jq -c 'select(.severity == "error" or .severity == "warn")'
   ```

2. **Pick the relevant error and grab its `fields.executionId`.** This is the run that failed. If the entry has only `activationId` and no `executionId`, scope to that instead.

3. **Pull the full trace for that one execution:**

   ```bash
   versori projects logs --environment production --since 1h --limit 1000 \
     --search 01KS2T...
   ```

   Lines come back in ascending time order — read top-to-bottom. The failure is at or near the bottom.

4. **For each line, identify:**

   - **Which task** emitted it (look for `taskId` in fields, or infer from the message).
   - **What input** went out (the "request" / "payload" log line just before the error — most workflows log the outbound payload).
   - **What the upstream replied** (the "response" / "status" line, or a runtime HTTP error with status code + body).
   - **Failure category**: 4xx (input/auth/payload), 5xx (upstream broken), in-workflow parse / mapping bug, or unhandled exception.

5. **Report back with three concrete pieces** before proposing a fix:

   - **Where** — task and timestamp: _"`upsert_product` failed at 16:07:42 (executionId `01KS2T...`)."_
   - **Why** — proximate cause from the log: _"Mirakl returned 400 with body `attribute 'Brand' is required`."_
   - **What input caused it** — the offending data: _"The product payload had `attributes: []` — no Brand attribute mapped."_

   Only after that diagnosis, propose a code change.

If logs are empty for the window, the workflow has not been triggered yet — confirm with the user that the source event has actually fired before assuming a runtime issue. If logs show only `info` lines and no error, the failure may be platform-side (deploy mis-configured, connection invalid, dynamic-variable missing) rather than in workflow code.

### `versori notifications channels list`

List notification channels in the current organisation. Use this before creating a new channel to avoid duplicates.

### `versori notifications channels create --name <name> --email <addr> [--cc <addr>]...`

Create an email notification channel for the current organisation. `--email` sets the primary recipient and must be supplied — service-key tokens carry no user identity, so there is no auto-derivation from the active context. Use `--cc` (repeatable) for additional recipients.

**Agent: ask the user for the recipient email address before invoking.** The CLI hard-exits if `--email` is missing — there is no picker fallback, no JWT lookup, and no env-var default.

Channels are organisation-scoped — create once, bind to as many projects/environments as needed.

### `versori notifications channels delete --channel-id <id> [--yes]`

Delete an org-scoped notification channel. Aliases: `rm`, `remove`. Omit `--channel-id` to get an interactive picker of existing channels by name. Confirms before deleting unless `--yes` is passed.

**Agent: always pass both `--channel-id` (from `versori notifications channels list -o json`) and `--yes`.**

**Project bindings using the deleted channel stop firing.** If you want a clean tear-down, unlink the bindings first with `versori notifications project unlink`.

### `versori notifications project list [--project <project-id>] [--environment <name>]`

List notification-channel bindings on a project. Optionally filter by environment name (e.g. `production`). `--project` defaults from `.versori` when inside a synced project directory.

### `versori notifications project link --channel-id <id> --environment <name> [--name <label>] [--project <project-id>]`

Link an existing channel to a project + environment. After linking, issues raised in that environment by workflow code (`ctx.createIssue()` or `.catch()` blocks) trigger an email through the linked channel.

If `--channel-id` is omitted, the CLI fetches the org's channels and shows an interactive picker (label format: `<channel-name>  (<to-address>)`). If `--environment` is omitted and the project has more than one environment, the CLI shows an environment picker (single-env projects auto-select). `--name` defaults to the channel's name.

**Agent: always pass `--channel-id` and `--environment` explicitly.** Source `--channel-id` from `versori notifications channels list -o json`; `--environment` is the human-readable env name (e.g. `production`).

```bash
versori notifications channels list
# → 01KS2TWXJYM...  ops-alerts  george@versori.com
versori notifications project link \
  --channel-id 01KS2TWXJYM... \
  --environment production \
  --name "ops-alerts (production)"
# → Linked channel "ops-alerts" to environment "production" on project 01KRR... .
```

### `versori notifications project unlink --notification-id <id> --environment <name> [--project <project-id>] [--yes]`

Remove a project-notification binding (stops alerts; the channel itself stays). Aliases: `rm`, `delete`. Omit `--notification-id` to pick an existing binding from a list by name. Confirms before deleting unless `--yes` is passed. The channel is not deleted; remove it separately with `versori notifications channels delete`.

**Agent: always pass `--notification-id`, `--environment`, and `--yes`.** Source `--notification-id` from `versori notifications project list --project <id> -o json` (the binding's `id`); `--environment` is the human-readable env name.

```bash
versori notifications project unlink
# → prompts for env, then for binding to remove, then confirms
versori notifications project unlink --notification-id 01KS2TX49C... --environment production --yes
```

Typical setup for issue-driven email alerts:

```bash
versori notifications channels list                                # check for an existing channel
versori notifications channels create --name ops-alerts --email ops@yourco.example  # --email is required (ask the user)
versori notifications project link \
  --channel-id <id from channels list> \
  --environment production \
  --name "ops-alerts (production)"                                 # routes ctx.createIssue() → email
versori notifications project list                                  # verify the binding
```

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

`versori projects sync` writes a JSON `.versori` file into the synced directory containing the project's `project_id` and the `context` active at sync time. Any command that accepts `--project` will read it from `.versori` when the flag is omitted.

**Resolution rules** (apply uniformly to every `--project` command):

| State of cwd / flag | Behavior |
|---|---|
| No `.versori` in cwd | `--project` is required; commands without it error out. |
| `.versori` present, `--project` omitted | `.versori`'s `project_id` is used. |
| `.versori` present, `--project` matches | The flag is used; no warning. |
| `.versori` present, `--project` differs | **`--project` wins**; a `warning: --project … overrides .versori project …` line is written to stderr. |
| `.versori.context` differs from the active CLI context | The **active context wins**; a `warning: active context … overrides .versori context …` line is written to stderr. The project_id is still used (a re-synced project keeps its ULID across contexts). |

Warnings go to stderr, so JSON / piped output on stdout stays clean.

**Destructive commands have an additional gate.** For `versori projects deploy`, `save`, and `sync`, the stderr warning above isn't enough on its own — those commands write project state and a missed warning can mean uploading the wrong source into a remote project (or wiping local files for a re-pin). When `--project` differs from the dir's `.versori`, these commands run an interactive confirmation step before proceeding. **Agents must always pass `--confirm`** on `deploy` / `save` / `sync` — the confirmation is human-only and blocks on stdin in non-interactive contexts. The "dir" here is the resolved `--directory` (defaults to cwd), so `--directory <path>` shifts the check to `<path>/.versori`. Read-only commands (`assets list`, `systems list`, `variables list`, `logs`, etc.) keep the stderr-warning-only behaviour.

**Agent: before invoking any project-scoped command, make the intended local project directory your cwd when local files or `.versori` defaults matter.** This is especially important for `deploy`, `save`, `sync`, logs, assets, systems, variables, activations, and notification project links. One-liner check:

```bash
cat .versori 2>/dev/null || echo '(no .versori in cwd)'
```

If `.versori` disagrees with the project the user asked about, `cd` to the correct synced directory before running the command, or pass the command's explicit `--directory` / `-d` flag and treat that directory as the source of truth. Do not rely on `--project` alone to compensate for being in the wrong project directory: it changes the remote project ID, but it does not change which local files are deployed or which target directory a command inspects.

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
