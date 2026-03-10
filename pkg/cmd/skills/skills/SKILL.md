---
name: coding-versori-sdk
description: Use this skill whenever the user wants to create, debug, or modify data integration workflows using the versori-run SDK. Triggers include requests to build ETL processes, API integrations, data transformation pipelines, database synchronization, webhooks, file processing, real-time data streaming, or any data integration service. Use when the user mentions Versori, versori-run, or needs help with TypeScript-based integration code.
---

# Versori Integration Skill

Expert-level data integration code using the versori-run SDK.

## Core Principles

**TypeScript only.** Do not generate code in any other language. If asked, explain that versori-run requires TypeScript.

**Scope validation.** Decline requests unrelated to data integration (ETL, API integrations, database sync, data transformation, file processing, webhooks, real-time streaming). Politely explain what you specialise in.

## Workflow Pattern

```typescript
import { fn, durable, schedule, http, webhook, workflow } from '@versori/run';

const myWorkflow = schedule('every-minute', '* * * * *')
    .then(
        http('fetch-data', { connection: 'source-system' }, async ({ fetch, log, openKv, activation }) => {
            const kv = openKv();
            const lastSync = await kv.get<string>('lastSync');
            const storeId = activation.getVariable('storeId') as string;

            log.info('Fetching data', { lastSync, storeId });

            const params = new URLSearchParams();
            if (lastSync) params.append('since', lastSync);

            // CRITICAL: Use PATH only, never full URLs
            const resp = await fetch(`/stores/${storeId}/data?${params}`);
            if (!resp.ok) throw new Error('Failed to fetch data');

            await kv.set('lastSync', new Date().toISOString());
            return await resp.json();
        })
    )
    .then(
        fn('process-data', ({ log, data }) => {
            log.debug('Processing data');
            return { processed: true, result: data };
        })
    )
    .then(
        http('send-data', { connection: 'target-system' }, async ({ fetch, data }) => {
            const resp = await fetch('/data', { method: 'POST', body: JSON.stringify(data) });
            if (!resp.ok) throw new Error('Failed to send data');
            return await resp.json();
        })
    );

async function main(): Promise<void> {
    const mi = await durable.DurableInterpreter.newInstance();
    mi.register(myWorkflow);
    await mi.start();
}

main().then().catch((err) => console.error('Failed to run main()', err));
```

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
  "scripts": {
    "build": "tsc",
    "start": "node dist/index.js"
  },
  "dependencies": {
    "@versori/run": "^0.4.0"
  },
  "devDependencies": {
    "typescript": "^4.9.0",
    "ts-node": "^10.9.0"
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

### HTTP Requests
- **Always** use `http()` tasks for API requests, never `fn()`
- **Always** use `ctx.fetch` (destructured as `fetch`) — never global `fetch` or NPM packages
- **Always** use the PATH only in fetch URLs, never full URLs
  - ✅ `fetch('/api/v1/resource')`
  - ❌ `fetch('https://api.example.com/api/v1/resource')`

### Connection Names
- After research, review the System & Authentication section for any systems that need user-specific configuration (e.g., shop domain, subdomain, instance URL). Ask the user for these values before proceeding. Then run `versori projects systems bootstrap --file <path> --project <id> --system-overrides '<json>'` (passing confirmed user-specific values via the overrides flag) to create systems, and run `versori projects systems list --project <id> --environment <env>` to verify what was created
- After verifying systems, create connections for each system using `versori connections create --project <id> --environment <env> --name <system-name> --template-id <template-id> --bypass` (use `--bypass` while connections are in active development)
- **Always** run `versori projects systems list` before generating workflow code if a project ID is known
- Use **exact** system names from the returned list — case-sensitive, no reformatting
  - ✅ `http('fetch', { connection: 'shopify' }, ...)` (if system is named `shopify`)
  - ❌ `http('fetch', { connection: 'Shopify' }, ...)`
- If a required system is **still missing after bootstrap**, stop and tell the user which systems are missing before writing any code. Ask for the name of their org, then give them the direct link `https://ai.versori.com/integrations/<project-id>?org=<org>` to add the missing systems. Proceed once they confirm.

### Task Types
- `http()` — API requests (requires a `connection`)
- `fn()` — data processing, transformation, business logic only

### KV Store
Only use for data that persists between executions AND is both SET and READ in the workflow.
See `references/kv-store.md` for patterns, API details, and anti-patterns.

### Durable Workflows
DurableInterpreter is the default choice for production workflows. Use MemoryInterpreter only for development, testing, or workflows that don't need persistent KV.
See `references/durable.md` for structure and options.

## Trigger Variations

```typescript
schedule('daily-sync', '0 0 * * *')       // Daily at midnight
schedule('hourly', '0 * * * *')           // Every hour
schedule('every-5-min', '*/5 * * * *')    // Every 5 minutes
webhook('stripe-webhook', '/webhooks/stripe')
```

## Integration Variables

```typescript
const apiKey = activation.getVariable('apiKey') as string;
activation.setVariable('lastSyncTime', new Date().toISOString());
```

## CLI Commands

Use the `versori` CLI when the user wants to list projects, create projects, pull down existing code, switch contexts, bootstrap systems, create connections, manage project assets, or deploy.
See `references/cli-usage.md` for all commands, options, deployment safety guidelines, and the recommended `.gitignore`.

**Always confirm before deploying or bootstrapping** unless the user explicitly says "deploy", "ship it", or "go ahead".

**Always dry-run before syncing** — `sync` deletes local files not present in the platform. Show the user the diff and confirm before running for real.

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

After confirming any required values, run `versori projects systems bootstrap --file versori-research/research.md --project <id> --system-overrides '{"Shopify": {"base_url": "https://my-store.myshopify.com/admin/api/2024-01"}}'` to create the required systems in the project from the research file, then verify with `versori projects systems list --project <id> --environment <env>`. Next, create connections for each system using `versori connections create ... --bypass` before proceeding to code generation. Omit `--system-overrides` if no systems require user-specific configuration.

After bootstrapping, upload the research document as a project asset using `versori projects assets upload --file versori-research/research.md --project <id>` so it is available as context for Versori AI agents.

See `references/research-docs.md` for the required document structure, inclusions, and exclusions.
