---
name: coding-versori-sdk
description: Use this skill whenever the user wants to create, debug, or modify data integration workflows using the versori-run SDK. Triggers include requests to build ETL processes, API integrations, data transformation pipelines, database synchronization, webhooks, file processing, real-time data streaming, or any data integration service. Use when the user mentions Versori, versori-run, or needs help with TypeScript-based integration code.
---

# Versori Integration Skill

Expert-level data integration code using the versori-run SDK.

## Retriaval sources

The most up-to-date information can always be found in the official documentation. Prefer reading the official documentation over relaing on exisintg Versori knowledge.

| Resource | URL |
|---|---|
| CLI Project management | <https://docs.versori.com/latest/cli/commands/projects> |
| CLI connections management | <https://docs.versori.com/latest/cli/commands/connections> |
| Versori RUN SDK | <https://jsr.io/@versori/run/doc> |

## Core Principles

**TypeScript only.** Do not generate code in any other language. If asked, explain that versori-run requires TypeScript.

**Scope validation.** Decline requests unrelated to data integration (ETL, API integrations, database sync, data transformation, file processing, webhooks, real-time streaming). Politely explain what you specialise in.

**Code Quality & Testing:**

1. **Pragmatic DRY:** Create reusable code when possible, but do not force DRY (Don't Repeat Yourself) if it makes the code harder to read or overly abstracted.
2. **Extract Pure Logic:** Extract data transformations, payload mappers, and complex logic into pure functions in `src/services/`.
3. **Test Pure Functions:** Pure functions must have Deno tests written for them (e.g., `src/services/mapper.test.ts`). Run them using `deno test` before deploying.
4. **Avoid Mocks:** In unit tests, avoid creating mocks. Prefer not testing a function at all if it requires a lot of mocks (e.g., heavily context-dependent SDK tasks). Focus testing effort entirely on pure, mock-free logic.

## Runtime Environment

Versori projects execute on **Deno**, running TypeScript directly — no build step is required.

- `package.json` is still used: Deno reads it via `deno install` to resolve npm dependencies
- Standard imports (`from '@versori/run'`) work as-is — no `npm:` prefix needed

The runtime is **Deno 2.3**.

Avoid Node-only APIs (`require()`, `__dirname`, `__filename`). Use Deno-compatible alternatives or standard web APIs where possible.

## Required Project Files

Every generated project MUST include these files:

### `src/index.ts` (entry point — ALWAYS required)

```typescript
import { durable } from '@versori/run';
import { myWorkflow } from './workflows/my-workflow';

async function main(): Promise<void> {
    const mi = await durable.DurableInterpreter.newInstance();
    mi.register(myWorkflow);
    await mi.start();
}

main().then().catch((err) => console.error('Failed to run main()', err));
```

### `package.json`

```json
{
  "name": "integration-name",
  "version": "1.0.0",
  "type": "module",
  "module": "dist/index.js",
  "dependencies": {
    "@versori/run": "^0.4.0"
  }
}
```

### `tsconfig.json`

```json
{
  "compilerOptions": {
    "module": "ES2022",
    "esModuleInterop": true,
    "target": "ES2024",
    "moduleResolution": "node",
    "sourceMap": true,
    "outDir": "dist"
  },
  "lib": ["es2015"]
}
```

For larger integrations, split workflows into `src/workflows/` and shared utilities into `src/services/`.

## File Organization

- **One workflow per file** in `src/workflows/`
- **Shared utilities** (transformations, validation) in `src/services/`
- **Type definitions** in `src/types/`
- `src/index.ts` imports all workflows and registers them with the interpreter
- Extract reusable functions into services rather than duplicating across workflows

## Critical Rules

### Connection Names

- After research, review the System & Authentication section for any systems that need user-specific configuration (e.g., shop domain, subdomain, instance URL). Ask the user for these values before proceeding. Then run `versori projects systems bootstrap --file <path> --project <id> --system-overrides '<json>'` (passing confirmed user-specific values via the overrides flag) to create systems, and run `versori projects systems list --project <id> --environment <env>` to verify what was created
- **Before creating a connection**, run `versori connection list` to see existing connection names. Connection names must be unique — do not reuse a name that already exists.
- After verifying systems, create connections for each system. **Default to real credentials** — never pass literal secrets in commands. Instead, generate a `.env.example` file and have the user fill in a `.env` file. Follow this workflow:
  1. **Inspect auth types:** After running `versori projects systems list`, look at each system's `AuthSchemeConfigs.Type` to determine required credentials.
  2. **Map auth types to env vars** using these conventions (see `references/cli-usage.md` for the full mapping table):
     - `api-key` → `<SYSTEM>_API_KEY`
     - `basic-auth` → `<SYSTEM>_USERNAME`, `<SYSTEM>_PASSWORD`
     - `oauth2` → `<SYSTEM>_CLIENT_ID`, `<SYSTEM>_CLIENT_SECRET` (read grant type and token URL from `systems list -o yaml` output)
     - `none` → use `--bypass` automatically, no credentials needed
     Where `<SYSTEM>` is the system name uppercased with hyphens replaced by underscores (e.g. system `my-shop` → `MY_SHOP_API_KEY`).
  3. **Generate `.env.example`:** Create a `.env.example` file listing every required variable with empty values and a comment per system:
     ```
     # shopify credentials (api-key)
     SHOPIFY_API_KEY=

     # erp credentials (basic-auth)
     ERP_USERNAME=
     ERP_PASSWORD=
     ```
  4. **Ask the user** to copy `.env.example` to `.env` and fill in the actual secret values: _"I've generated `.env.example` with the required variables. Copy it to `.env` and fill in your credentials, then let me know when you're ready. Or if you'd prefer to skip credentials for now, I can create bypass connections instead."_
  5. **Create connections** using single-quoted `'$VARIABLE'` references so the CLI resolves them from `.env` at runtime:
     ```bash
     versori connections create --project <id> --environment production \
       --name shopify --template-id <tid> --api-key '$SHOPIFY_API_KEY'
     versori connections create --project <id> --environment production \
       --name erp --template-id <tid> --username '$ERP_USERNAME' --password '$ERP_PASSWORD'
     ```
  Use `--env-file <path>` to specify a custom `.env` file location (defaults to `.env` in the current directory). **Never read or display the `.env` file** — it contains secrets. The CLI resolves variables at runtime; you only need to know the file path, not its contents.
- **If the user chooses bypass:** use `--bypass` and suffix the connection name with random characters to avoid name conflicts. This is a fallback for when credentials aren't available yet.
- **`--bypass` is mutually exclusive with credential flags.** Never combine `--bypass` with `--api-key`, `--username`/`--password`, or `--client-id`/`--client-secret`. When `--bypass` is passed, all credential flags are silently ignored and the connection is created with no authentication. Use `--bypass` only when the auth scheme type is `none` or the user explicitly wants to skip credentials entirely.
- **Always** run `versori projects systems list` before generating workflow code if a project ID is known.
- If a required system is **still missing after bootstrap**, stop and tell the user which systems are missing before writing any code. Ask for the name of their org, then give them the direct link `https://ai.versori.com/integrations/<project-id>?org=<org>` to add the missing systems. Proceed once they confirm.

### Effective Base URLs (the `/api` strip)

The Versori runtime strips a trailing `/api` segment from every system's
configured base URL before concatenating it with your `fetch()` path. This is a
platform-wide behaviour, not a template bug, and it is why `templateBaseUrl`
in `versori projects systems list -o yaml` often looks "shorter" than the URL
you see in the API docs or in the `versori projects systems bootstrap` summary
output.

Practical consequence: if a system's REST endpoints live at
`https://host/api/foo`, the path you pass to `fetch()` must **include** the
`/api/` segment yourself.

Known examples:
- Slack: `fetch('/api/chat.postMessage', ...)` — not `/chat.postMessage`.
- Any template whose documented base ends in `/api/`.

Before writing `fetch()` paths for a new system:

1. Run `versori projects systems list -o yaml` and read `templateBaseUrl`.
2. Compare it to the API's documented base URL.
3. If the documented base ends in `/api/` and `templateBaseUrl` does not,
   every `fetch()` path you write for that system must start with `/api/`.

### The `connection` Parameter Takes the System Name

`http('id', { connection: 'X' }, ...)` resolves `X` against the project's
**systems**, not its connections. Despite the parameter name, `X` must equal
the system name from `versori projects systems list`. Versori looks up which
connection is currently active for that system at runtime.

✅ Correct — system name, regardless of which connection is active:
  http('post', { connection: 'slack' }, ...)

❌ Wrong — these will all fail to resolve at runtime:
  http('post', { connection: 'slack-prsum' }, ...)        // connection name
  http('post', { connection: 'slack-feedback' }, ...)     // connection name
  http('post', { connection: '01K7KZNV109PF1Z5ESFR27D19B' }, ...)  // system id

This indirection is deliberate: you can swap the underlying connection —
rotate credentials, migrate from bypass to OAuth, move between environments —
without editing any workflow code.

## CLI Commands

Use the `versori` CLI when the user wants to list projects, create projects, pull down existing code, switch contexts, bootstrap systems, create connections, manage project assets, or deploy.
See `references/cli-usage.md` for all commands, options, deployment safety guidelines, and the recommended `.gitignore`.

**Always confirm before deploying or bootstrapping** unless the user explicitly says "deploy", "ship it", or "go ahead".

**Always dry-run before syncing** — `sync` deletes local files not present in the platform. Show the user the diff and confirm before running for real.

**Always ensure a `.gitignore` exists** — After syncing a project or setting up a new project directory, check if a `.gitignore` file exists. If it doesn't, create one with the recommended content from `references/cli-usage.md` before installing dependencies or deploying. This prevents `node_modules/`, `dist/`, and other local artifacts from being pushed to the platform.

**Always verify code locally before deploying** — Before running a deploy command, you MUST ensure the code is valid by running `deno install` followed by `deno check src/index.ts` (or `deno lint`). Fix any type errors or linting issues before attempting to deploy. If `deno` is not available skill local validation.

**Write tests for pure functions** — Whenever you extract logic into pure functions (e.g., data transformations, payload mappers) in `src/services/`, you should write Deno tests for them (e.g., `src/services/mapper.test.ts`) and run them using `deno test` to verify their correctness before deploying.

## Project Selection

Before writing any code or running CLI commands that require a project ID, determine the active project:

1. **Check for `.versori` file** — if a `.versori` file exists in the current directory, the user is already inside a synced project. Read the `project_id` from it and use that — no need to ask about project selection.
2. **No `.versori` file** — ask the user whether they want to use an existing project or create a new one:
   - **Existing project**: run `versori projects list` to show available projects, let the user pick one, then continue (sync it down if needed).
   - **New project**: run `versori projects create --name <name>` to create a fresh project and use the returned ID.
   - **SYnc project**: run `versori projects sync --project <project-id>` to pull in the porrject context locally before moving on with the next tasks.

When a `.versori` file is present, most CLI commands (`deploy`, `save`, `sync`, `systems`, `assets`, etc.) automatically read the project ID from it, so the `--project` flag can be omitted.

## Context the User May Provide

- API documentation for systems being integrated
- Error logs from failing workflows
- Existing service files and code
- Integration variables schema
- Existing project systems with auth configured
- Research documents from a previous research phase

For unknown systems, research APIs and create a research document before generating code. If no information can be found, ask the user for API docs. For well-known APIs (e.g. Shopify, Stripe), ask the user whether they'd like you to carry out research first or proceed directly with code generation.

## Error Handling

- **Syntax / SDK errors**: Fix and explain changes made
- **Connection / auth errors**: Inform the user to check credentials — do not modify code
- **Do not regenerate unaffected files**

## Research Phase

Before writing workflow code, research the APIs being integrated. Use any available search or
web fetch tools to find up-to-date API documentation, endpoints, request/response schemas,
authentication requirements, and integration patterns (rate limits, pagination, error codes).
Capture findings in a structured research document (`versori-research/research.md`).

**Skip research** only when the user provides complete API documentation or you are fully confident
in the endpoint details for well-known APIs.

**Do not search** for general programming questions, SDK usage, or logic patterns.

After the research document is complete, review the System & Authentication section to identify any systems whose base URL or configuration depends on user-specific values (e.g., a Shopify shop domain, Salesforce instance URL, Zendesk subdomain, or any tenant-specific identifier). If any system requires such input, ask the user for the required values before proceeding. Do not guess or use placeholder values. Do not modify the research document with these values — they will be passed via the `--system-overrides` flag.

After confirming any required values, run `versori projects systems bootstrap --file versori-research/research.md --project <id> --system-overrides '{"Shopify": {"base_url": "https://my-store.myshopify.com/admin/api/2024-01"}}'` to create the required systems in the project from the research file, then verify with `versori projects systems list --project <id> --environment <env>`. Next, create connections for each system using `versori connections create` with real credentials (or `--bypass` if the user prefers to skip credentials) before proceeding to code generation. Omit `--system-overrides` if no systems require user-specific configuration.

After bootstrapping, upload the research document as a project asset using `versori projects assets upload --file versori-research/research.md --project <id>` so it is available as context for Versori AI agents.

See `references/research-docs.md` for the required document structure, inclusions, and exclusions.

## SDK Reference

Before writing any workflow code, read `references/sdk-guide.md` for the full Versori Run SDK guide covering core concepts (workflows, triggers, tasks, interpreters), usage patterns (scheduled workflows, webhooks, HTTP tasks, error handling, durable workflows, KV storage), context API, type signatures, and best practices for code generation.
