# Versori Run SDK Guide

Reference for generating integration workflows with the Versori Run SDK. Covers core concepts, usage patterns, and best practices.

## Contents

- [Users and Activations](#users-and-activations)
- [Core Concepts](#core-concepts)
  - [Workflows](#workflows)
  - [Triggers](#triggers)
  - [Tasks](#tasks)
  - [Interpreters](#interpreters)
- [Basic Patterns](#basic-patterns)
  - [Starting a Project](#starting-a-project)
  - [Scheduled Workflow](#scheduled-workflow)
  - [Webhook Handler](#webhook-handler)
  - [HTTP Task with Authenticated Connection](#http-task-with-authenticated-connection)
  - [Error Handling](#error-handling)
  - [Accessing Credentials Outside an HTTP Task](#accessing-credentials-outside-an-http-task)
  - [Creating Issues](#creating-issues)
    - [Parameters](#parameters)
    - [Options](#options)
    - [Example](#example)
- [Durable Workflows](#durable-workflows)
  - [Defining a Durable Workflow](#defining-a-durable-workflow)
  - [Starting a Durable Workflow](#starting-a-durable-workflow)
    - [Workflow Object Properties](#workflow-object-properties)
  - [Rescheduling Workflows](#rescheduling-workflows)
  - [WorkflowClient Methods](#workflowclient-methods)
  - [Using KV Storage with Durable Workflows](#using-kv-storage-with-durable-workflows)
- [Context Object (`ctx`)](#context-object-ctx)
  - [Activation](#activation)
    - [Properties](#properties)
    - [Methods](#methods)
    - [User Type](#user-type)
  - [AsyncWorkflow](#asyncworkflow)
- [Key-Value Storage](#key-value-storage)
- [Common Patterns](#common-patterns)
  - [Data Transformation Pipeline](#data-transformation-pipeline)
  - [Error Handling with Issue Reporting](#error-handling-with-issue-reporting)
  - [Retry](#retry)
    - [Generic Retry Helper](#generic-retry-helper)
    - [HTTP Retry with Timeout and Abort](#http-retry-with-timeout-and-abort)
- [HttpContext (for `http` Tasks)](#httpcontext-for-http-tasks)
- [Webhook Options Reference](#webhook-options-reference)
- [Quick Reference](#quick-reference)
- [Type Signatures](#type-signatures)
- [Best Practices](#best-practices)

## Users and Activations

A **user** represents an external user on the Versori platform. Users hold credentials for external systems.

Each user is linked to an **activation** — a specification of which credentials to use for a given external system. Every workflow runs in the context of an activation, so credentials are always available.

By default the platform provides a **static user**. When no external users exist, all connections use the static user. From a workflow's perspective there is no difference between static and dynamic users.

## Core Concepts

### Workflows

A workflow is a sequence of tasks started by a trigger (schedule, webhook, or another workflow).

### Triggers

Triggers determine how a workflow starts:

| Trigger | Function | Description |
|---------|----------|-------------|
| Schedule | `schedule(id, cron, activationPredicate?)` | Runs on a cron schedule |
| Webhook | `webhook(id, options?)` | Triggered by an HTTP request |
| Durable Workflow | `workflow(id, options?)` | Started programmatically via `ctx.start()` or `ctx.workflowClient().startWorkflow()` |

### Tasks

Tasks are units of work within a workflow:

| Task | Function | Description |
|------|----------|-------------|
| Function | `fn(id, callback)` | Generic task with context access |
| HTTP | `http(id, {connection}, callback)` | HTTP request with automatic authentication |

Tasks can be chained with `.then()` and `.catch()`.

### Interpreters

Interpreters execute workflows:

| Interpreter | Use Case |
|-------------|----------|
| `MemoryInterpreter` | Development, testing, and simple deployments. Suitable for workflows that do not need persistent KV or durable execution. |
| `DurableInterpreter` | Production use. Provides durable workflow execution and database-backed KV storage. Preferred for almost all cases. |

---

## Basic Patterns

### Starting a Project

Import the `durable` module and create a `DurableInterpreter`. Register workflows, then start the interpreter. Keep workflow implementations in separate files under the `workflows` folder.

```typescript
import { durable } from '@versori/run';
import { processDataWorkflow } from './workflows/durable-example';

async function main(): Promise<void> {
  const mi = await durable.DurableInterpreter.newInstance();

  mi.register(processDataWorkflow);

  await mi.start();
}

main().then().catch((err) => console.error('Failed to run main()', err));
```

### Scheduled Workflow

The `schedule` trigger runs a workflow on a cron schedule. The optional `'all'` activation predicate runs the workflow for every user; omitting it runs only for the static user.

```typescript
import { fn, schedule } from '@versori/run';

const dailySync = schedule('daily-sync', '0 0 * * *', 'all')
  .then(
    fn('sync-data', async (ctx) => {
      ctx.log.info('Starting sync');
      return { synced: true };
    })
  );
```

### Webhook Handler

The `webhook` trigger creates an HTTP endpoint on the Versori platform.

**Options:**

| Option | Description |
|--------|-------------|
| `method` | HTTP method for the endpoint. Defaults to `'all'`. |
| `cors` | CORS configuration (see below). |
| `response` | Response mode: `'sync'` (wait for completion) or `'async'` (return immediately). Defaults to `'sync'`. |
| `connection` | Connection used for verifying webhook signatures or authentication. Supports `api-key`, `basicauth`, and `hmac`. |
| `request.rawBody` | When `true`, the raw body is passed instead of parsed JSON. Access it via `ctx.request()` — `ctx.data` will be `undefined`. |

**CORS options:**

| Option | Description |
|--------|-------------|
| `origin` | Allowed origin |
| `methods` | Allowed HTTP methods |
| `allowedHeaders` | Allowed request headers |
| `exposedHeaders` | Headers exposed in the response |
| `credentials` | Whether to allow credentials |

```typescript
import { fn, webhook } from '@versori/run';

const handleWebhook = webhook('incoming-data', {
  method: 'post',
  response: { mode: 'sync' }
})
  .then(
    fn('process', async (ctx) => {
      const payload = ctx.data;
      ctx.log.info('Received', { payload });
      const rawRequest = ctx.request();  // Returns express.Request or undefined
      return { processed: true };
    })
  );
```

### HTTP Task with Authenticated Connection

The `http` task sends requests through a named connection. The connection provides automatic authentication via credentials configured in the Versori platform — connection names are set outside the code.

```typescript
import { fn, http, schedule } from '@versori/run';

const fetchFromAPI = schedule('fetch-api', '*/5 * * * *')
  .then(
    http('get-data', { connection: 'my-api' }, async ({ fetch, log, data }) => {
      const response = await fetch('/users');
      const users = await response.json();
      log.info('Fetched users', { count: users.length });
      return users;
    })
  );
```

### Error Handling

Use `.catch()` to handle errors at the workflow level. Inside a catch block, `ctx.data` contains the thrown exception. Prefer `.catch()` over `try/catch` for observability; reserve `try/catch` within tasks for adding context or performing cleanup before re-throwing.

Do not silently swallow errors unless explicitly required.

```typescript
import { fn, http, schedule } from '@versori/run';

const handleErrors = schedule('fetch-api', '*/5 * * * *')
  .then(
    http('get-data', { connection: 'my-api' }, async ({ fetch, log, data }) => {
      const response = await fetch('/users');
      const users = await response.json();
      log.info('Fetched users', { count: users.length });
      return users;
    })
  )
  .catch(
    fn('handle-error', async (ctx) => {
      ctx.log.error('Error fetching users', { error: ctx.data });
      return { error: ctx.data };
    })
  );
```

### Accessing Credentials Outside an HTTP Task

Use `ctx.credentials()` when you need authentication that is not handled by the `http` task (e.g., non-HTTP protocols or custom auth flows).

| Method | Description |
|--------|-------------|
| `getRaw(name, activationId?)` | Returns raw credential data as `Uint8Array`. Applicable for API Key credentials. |
| `getAccessToken(name, forceRefresh?, activationId?)` | Returns a token object (`accessToken`, `tokenType`, optional `expiry`). Works for OAuth 2.0, Basic, and Bearer authentication. Pass `true` for `forceRefresh` to force a token refresh. |
| `getOAuth1Metadata(name, activationId?)` | Returns OAuth1 authorization metadata. |

The connection name must match a connection configured in `versori.yaml`.

```typescript
import { fn, webhook } from '@versori/run';

webhook('example')
  .then(
    fn('use-credentials', async (ctx) => {
      const raw = await ctx.credentials().getRaw('my-api-key-connection');
      const apiKey = new TextDecoder('utf-8').decode(raw);

      const token = await ctx.credentials().getAccessToken('my-oauth-connection');
      console.log(token.accessToken);

      const freshToken = await ctx.credentials().getAccessToken('my-oauth-connection', true);
    })
  );
```

### Creating Issues

Issues are notifications surfaced in the Versori platform. They can trigger email alerts and are available for inspection in the UI.

Issues are created **automatically** when a `.catch()` block executes. You can also create them **manually** with `ctx.createIssue()` from any task.

**Important:** When deduplication is disabled, never create issues inside a loop — each issue can trigger multiple emails.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `severity` | `'low' \| 'medium' \| 'high'` | Yes | Severity level |
| `title` | `string` | Yes | Short title |
| `message` | `string` | Yes | Detailed description |
| `annotations` | `Record<string, string>` | Yes | Key-value metadata |

#### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `deduplicate` | `boolean` | `true` | Group issues with the same severity and title instead of creating duplicates |

#### Example

```typescript
import { fn, webhook } from '@versori/run';

const workflow = webhook('process-data')
  .then(
    fn('process', async (ctx) => {
      if (!ctx.data.valid) {
        await ctx.createIssue({
          severity: 'high',
          title: 'Data Processing Failed',
          message: `Error: ${ctx.data.message}`,
          annotations: {
            workflow: 'process-data',
            errorType: ctx.data.name || 'Unknown',
            timestamp: new Date().toISOString()
          }
        });
      }
      return { success: true };
    })
  )
  .catch(
    fn('handle-error', async (ctx) => {
      ctx.log.error('Processing failed', { error: ctx.data.message });
      return { error: ctx.data.message };
    })
  );
```

---

## Durable Workflows

Durable workflows run asynchronously and support retries, rescheduling, and cross-workflow orchestration. They are started programmatically from other workflows and execute with configurable retry policies.

### Defining a Durable Workflow

Use the `workflow` trigger. Options:

| Option | Type | Description |
|--------|------|-------------|
| `ttl` | `number` | Time-to-live in seconds — how long a single attempt can run before timing out |
| `limit` | `number` | Maximum number of concurrent workflow instances |
| `maxAttempts` | `number` | Maximum execution attempts. Prefer setting `maxAttempts` on `ctx.start()` instead. |

```typescript
import { fn, workflow } from '@versori/run';

const myWorkflow = workflow('my-workflow', { ttl: 60 })
  .then(
    fn('process', async (ctx) => {
      return { success: true };
    })
  );
```

### Starting a Durable Workflow

Start a durable workflow with `ctx.start()` or `ctx.workflowClient().startWorkflow()`. Both accept a workflow ID and an options object.

| Option | Description |
|--------|-------------|
| `data` | JSON-serializable data passed to the workflow |
| `dataRaw` | Base64-encoded data (mutually exclusive with `data`) |
| `maxAttempts` | Maximum retry attempts |

The returned workflow object can be used to check status or wait for completion.

```typescript
import { fn, webhook } from '@versori/run';

export const batchStartWebhook = webhook('batch-start', { response: { mode: 'sync' } })
  .then(
    fn('start-batch-workflows', async ({ data, log, openKv, workflowClient, executionId }) => {
      const wfClient = workflowClient();
      const wf = await wfClient.startWorkflow('process-data', {
        data: {},
        maxAttempts: 1_000_000
      });

      return {
        status: 'submitted',
        workflow: wf,
      };
    })
  );
```

#### Workflow Object Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | `string` | Unique identifier |
| `projectId` | `string` | Project this workflow belongs to |
| `environmentId` | `string` | Environment this workflow belongs to |
| `group` | `string` | Workflow group name |
| `payload` | `string` | Input data (base64-encoded) |
| `output` | `string?` | Output data (base64-encoded) |
| `status` | `'available' \| 'locked' \| 'completed' \| 'failed'` | Current status |
| `attempt` | `number` | Current attempt number |
| `maxAttempts` | `number?` | Maximum retry attempts |
| `timeout` | `string` | When the message will be redelivered |
| `createdAt` | `string` | Creation timestamp |
| `metadata` | `WorkflowMsgMetadata` | Contains `executionId`, `activationId`, `userId`, and optional `parentWorkflowId` |

### Rescheduling Workflows

A durable workflow can reschedule itself for iterative processing — useful for batch jobs that process items one at a time with delays between each.

```typescript
import { fn, workflow } from '@versori/run';

export const processDataWorkflow = workflow('process-data', { ttl: 60 })
  .then(
    fn('validate-input', async ({ log, openKv, executionId, workflowClient, workflow }) => {
      log.info('Validating input data', { executionId, workflow });

      const kv = await openKv();
      const index = await kv.get(`${executionId}/index`);
      const item = await kv.get(`${executionId}/items/${index.current}`);

      processItem(log, item);

      index.current++;
      await kv.set(`${executionId}/index`, index);
      log.info('Updated index', index);

      const wfClient = workflowClient();
      if (index.current < index.total) {
        log.info('Rescheduling workflow', { workflow });
        await wfClient.rescheduleWorkflow(workflow, '5s');
      } else {
        log.info('Completing workflow', { workflow });
        await wfClient.completeWorkflow(workflow, 'completed', {});
      }

      return {
        status: 'success',
        message: `Successfully processed ${index.current} items`,
        completedAt: new Date().toISOString()
      };
    })
  )
  .catch(async ({ error, log, data }) => {
    log.error('Workflow failed - will retry automatically', {
      error: error.message,
      attempt: data.attempt || 1
    });
    throw error;
  });
```

### WorkflowClient Methods

| Method | Description |
|--------|-------------|
| `startWorkflow(group, opts)` | Start a new workflow |
| `getWorkflow(workflow)` | Get the current state of a workflow |
| `getWorkflowById(id)` | Get a workflow by ID |
| `waitForCompletion(workflow)` | Poll until the workflow status is `'completed'` or `'failed'` |
| `rescheduleWorkflow(workflow, delay)` | Reschedule the workflow after a delay (e.g., `'5s'`, `'1m'`) |
| `completeWorkflow(workflow, status, result)` | Mark the workflow as `'completed'` or `'failed'` with a result object |

### Using KV Storage with Durable Workflows

Durable workflows commonly use KV storage to track progress across reschedules. Use `executionId` as a namespace to isolate data per execution.

```typescript
const kv = await openKv();

await kv.set(`${executionId}/index`, { current: 0, total: items.length });
await kv.set(`${executionId}/items/0`, items[0]);

const index = await kv.get(`${executionId}/index`);
```

---

## Context Object (`ctx`)

Every task receives a context object:

```typescript
interface Context<D> {
  data: D;                    // Input from the previous task
  log: Logger;                // Structured logging
  executionId: string;        // Unique execution ID
  startTime: Date;            // Execution start time
  activation: Activation;     // Current activation (user context)
  workflow?: Workflow;        // Workflow object (durable workflows only)

  openKv(scope?): KeyValue;           // Key-value storage
  credentials(): CredentialsProvider;  // Access credentials directly
  createIssue(issue, options?): Promise<Issue | null>;  // Create an issue/alert
  start(workflowId, opts): Promise<AsyncWorkflow>;      // Start a durable workflow (see AsyncWorkflow below)
  request(): express.Request | undefined;               // Raw HTTP request (webhooks only)
  workflowClient(): WorkflowInterface;                  // Workflow client for starting, getting, and rescheduling workflows
  destroy(scope): Promise<void>;                        // Destroy a KV store for a given scope
}
```

### Activation

The `ctx.activation` holds information about the current user's activation — which credentials and configuration apply to this execution.

```typescript
interface Activation {
  id: string;
  user: User;
  getVariable(name: string): unknown;
  setVariable(name: string, value: unknown): Promise<void>;
}
```

#### Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | `string` | Unique activation identifier |
| `user` | `User` | The user who owns this activation |

#### Methods

**`getVariable(name: string): unknown`**
Retrieves a dynamic variable for this activation. Returns `undefined` if the variable does not exist.

```typescript
const lastSyncDate = ctx.activation.getVariable('lastSyncDate') as string | undefined;
const settings = ctx.activation.getVariable('settings') as boolean | undefined;
```

**`setVariable(name: string, value: unknown): Promise<void>`**
Persists a dynamic variable to the platform and updates the local cache.

```typescript
await ctx.activation.setVariable('lastSyncDate', new Date().toISOString());
await ctx.activation.setVariable('settings', true);
```

#### User Type

```typescript
interface User {
  id: string;
  externalId: string;
  displayName: string;
  organisationId: string;
  createdAt: string;
  updatedAt: string;
}
```

### AsyncWorkflow

`ctx.start()` returns an `AsyncWorkflow` that tracks a durable workflow started from the current context:

| Property / Method | Type | Description |
|-------------------|------|-------------|
| `id` | `string` | Unique identifier for the workflow instance |
| `isCompleted` | `boolean` | Whether the workflow has finished |
| `isWaiting` | `boolean` | Whether a `wait()` call is in progress |
| `wait(maxWait?)` | `Promise<void>` | Poll until the workflow completes |
| `getData()` | `Promise<unknown>` | Get the parsed output data |
| `getDataRaw()` | `Promise<string>` | Get the raw output as a base64-encoded string |

---

## Key-Value Storage

KV storage persists data across executions. Scope determines visibility:

| Scope | Visibility |
|-------|------------|
| `':execution:'` | Current execution only |
| `':project:'` | All executions in the project |
| `':organization:'` | All executions in the organization |

```typescript
const kv = ctx.openKv(':project:');

await kv.set(['users', id], userData);
const user = await kv.get(['users', id]);
await kv.delete(['users', id]);
const list = await kv.list(['users']);
const count = await kv.count(['users']);
```

`kv.get()` accepts options to control missing-key behavior:

- `{ default: value }` — return a default value instead of `undefined` when the key is missing
- `{ throwOnNotFound: true }` — throw a `KVNotFoundError` when the key is missing

---

## Common Patterns

### Data Transformation Pipeline

```typescript
import { fn, http, schedule } from '@versori/run';

const pipeline = schedule('etl-pipeline', '0 */6 * * *')
  .then(
    http('fetch-source', { connection: 'source-api' }, async ({ fetch }) => {
      const res = await fetch('/data');
      return res.json();
    })
  )
  .then(
    fn('transform', (ctx) => {
      return ctx.data.map(item => ({
        ...item,
        transformed: true,
        timestamp: new Date().toISOString()
      }));
    })
  )
  .then(
    http('save-dest', { connection: 'dest-api' }, async ({ fetch, data }) => {
      await fetch('/bulk', {
        method: 'POST',
        body: JSON.stringify(data)
      });
      return { saved: data.length };
    })
  );
```

### Error Handling with Issue Reporting

```typescript
import { fn, webhook } from '@versori/run';

const robustWorkflow = webhook('process')
  .then(
    fn('risky-operation', async (ctx) => {
      if (!ctx.data.valid) {
        ctx.log.error('Invalid input', { data: ctx.data, error: 'Invalid input' });
        throw new Error('Invalid input');
      }
      return { success: true };
    })
  )
  .catch(
    fn('handle-error', async (ctx) => {
      ctx.log.error('Operation failed', { error: ctx.data.message });
      await ctx.createIssue({
        severity: 'high',
        title: 'Workflow Error',
        message: ctx.data.message,
        annotations: { workflow: 'process' }
      });
      return { success: false, error: ctx.data.message };
    })
  );
```

### Retry

There are no built-in retries outside of durable workflows. For non-durable workflows, implement retry logic manually — typically with exponential backoff.

#### Generic Retry Helper

```typescript
async function retryCall<T>(
  fn: () => Promise<T>,
  maxRetries: number
): Promise<T> {
  let lastError: Error | undefined;

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error));

      if (attempt === maxRetries) {
        break;
      }

      const delayMs = Math.pow(2, attempt) * 100;
      await new Promise((resolve) => setTimeout(resolve, delayMs));
    }
  }

  throw lastError;
}
```

#### HTTP Retry with Timeout and Abort

```typescript
type FetchFunction = typeof fetch;
type LogFunction = { info: (msg: string, data?: object) => void };

async function fetchWithRetry(
  url: string,
  options: RequestInit = {},
  {
    maxRetries = 3,
    timeoutMs = 5000,
    retryDelayMs = 1000,
    fetch: fetchFn = fetch,
    log,
    signal: externalSignal,
  }: {
    maxRetries?: number;
    timeoutMs?: number;
    retryDelayMs?: number;
    fetch?: FetchFunction;
    log?: LogFunction;
    signal?: AbortSignal;
  } = {},
): Promise<Response> {
  let lastError: Error | null = null;

  for (let attempt = 0; attempt < maxRetries; attempt++) {
    const signals = [AbortSignal.timeout(timeoutMs)];
    if (externalSignal) signals.push(externalSignal);

    const signal = AbortSignal.any(signals);

    try {
      const response = await fetchFn(url, { ...options, signal });

      if (response.status >= 500) {
        throw new Error(`Server error: ${response.status}`);
      }

      return response;
    } catch (error) {
      lastError = error as Error;

      if (externalSignal?.aborted) throw error;

      const isTimeout = (error as Error).name === 'AbortError';
      const isServerError = (error as Error).message?.includes('Server error');

      if ((!isTimeout && !isServerError) || attempt === maxRetries - 1) {
        throw error;
      }

      const delay = retryDelayMs * Math.pow(2, attempt);
      log?.info(`Attempt ${attempt + 1} failed, retrying in ${delay}ms...`, { attempt, delay });
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }

  throw lastError ?? new Error('Max retries exceeded');
}

// Usage within an http task
http('fetch-data', { connection: 'my-api' }, async (ctx) => {
  const response = await fetchWithRetry(
    '/users',
    { method: 'GET' },
    {
      fetch: ctx.fetch,
      log: ctx.log,
      maxRetries: 5,
      timeoutMs: 10000,
    },
  );
  return response.json();
});
```

---

## HttpContext (for `http` Tasks)

The `http` task extends the standard context with HTTP-specific properties:

```typescript
http('task-id', { connection: 'api-name' }, async (ctx) => {
  // ctx.fetch - authenticated fetch with base URL
  // ctx.baseUrl - the API base URL (Promise<string>)
  // ctx.connectionVariables - variables configured for this connection
  // ctx.pageParams - pagination parameters (if using pagination)
  // ctx.nextPage(params) - trigger next page (if using pagination)
  // Plus all standard Context properties
});
```

---

## Webhook Options Reference

```typescript
webhook('id', {
  method: 'post' | 'get' | 'put' | 'delete' | 'patch' | 'options' | 'head' | 'all',
  cors: true | { origin, methods, allowedHeaders, credentials },
  response: {
    mode: 'sync' | 'async',
    onSuccess: (ctx) => Response,
    onError: (ctx) => Response
  },
  connection: 'connection-name',
  request: { rawBody: true }
});
```

---

## Quick Reference

| Action | Code |
|--------|------|
| Log info | `ctx.log.info('msg', { data })` |
| Log error | `ctx.log.error('msg', { error })` |
| Get input data | `ctx.data` |
| Store data | `ctx.openKv(':project:').set(key, value)` |
| Get stored data | `ctx.openKv(':project:').get(key)` |
| Create issue | `ctx.createIssue({ severity, title, message, annotations })` |
| Start workflow | `ctx.start('workflow-id', { data, maxAttempts })` |
| HTTP fetch | `ctx.fetch('/path')` (in http task) |
| Access request | `ctx.request()` (in webhook) |

---

## Type Signatures

```typescript
function fn<In, Out>(id: string, callback: ContextFunc<In, Out>): Task<In, Out>;
function http<In, Out, PageParams>(id: string, opts: HttpOptions<PageParams>, callback: HttpContextFunc<In, Out, PageParams>): Task<In, Out>;

function schedule(id: string, cron: string, activationPredicate?: ActivationPredicate): Workflow<ScheduleData>;
function webhook(id: string, options?: WebhookOptions): Trigger<WebhookData>;
function workflow(id: string, options?: DurableWorkflowOptions): Trigger<DurableWorkflowData>;

type ActivationPredicate = 'all' | ((a?: Activation) => boolean);
type ContextFunc<In, Out> = (ctx: Context<In>, idx?: number) => Out | Promise<Out>;

interface KeyValue {
  get<T>(key: string | string[], options?: GetOptions<T>): Promise<T | undefined>;
  set<T>(key: string | string[], value: T): Promise<void>;
  delete(key: string | string[]): Promise<void>;
  list(prefix: string[], options?: ListKVRequest): Promise<ListKVResponse>;
  count(prefix: string[]): Promise<CountKVResponse>;
}
```

---

## Best Practices

1. **Give tasks meaningful IDs** — they appear in execution traces and logs.
2. **Use structured logging** — `ctx.log.info('action', { key: value })`. Use a static message string and pass dynamic data as structured arguments for searchability. Never template the log message. Never use emojis or colors in log messages.
3. **Handle errors with `.catch()`** — improves observability when workflows fail.
4. **Return data from tasks** — the next task receives it via `ctx.data`.
5. **Use `http` for API calls** — automatic authentication via connections.
6. **Use `fn` for data processing** — pure logic without external calls.
