# Versori Run SDK - Guide for LLM Agents

This guide helps LLM agents understand and generate code using the Versori Run SDK, an integration development framework for building production-ready workflows.
The guide focuses on core features of the SDK, describing when and how to use them.
It also provides common patterns for agents to use when generating code.

## Users

Users are a concept in the Versori platform that represent an external user in an organization. Users hold credentials to external systems. For the most part, users are hidden in the SDK.

Each user in the Versori platform is linked to an activation. An activation is a specification of which credentials to use for an external system. When a workflow is started it is started in the context of an activation. There will always be an activation available in the context of a workflow.

By default the Versori platform has a static user. If there are no external users all connections are created with the static user. So from a workflow's perspective there is no difference between a workflow with a static user and a workflow with a dynamic user.

## Core Concepts

### 1. Workflows

A workflow is a sequence of tasks triggered by an event (schedule, webhook, workflow).

### 2. Triggers

Triggers determine **how** a workflow starts:

| Trigger | Function | Description |
|---------|----------|-------------|
| Schedule | `schedule(id, cron, activationPredicate?)` | Runs on a cron schedule |
| Webhook | `webhook(id, options?)` | Triggered by HTTP requests |
| Durable Workflow | `workflow(id, options?)` | Started programmatically via `ctx.workflowClient().startWorkflow()` |

### 3. Tasks

Tasks are units of work that process data:

| Task | Function | Description |
|------|----------|-------------|
| Function | `fn(id, callback)` | Generic task with context access |
| HTTP | `http(id, {connection}, callback)` | HTTP requests with auto-auth |

### 4. Interpreters

Interpreters execute workflows:

| Interpreter | Use Case |
|-------------|----------|
| `MemoryInterpreter` | Development, testing, simple deployments. Used for examples and workflows that don't need persistent KV. |
| `DurableInterpreter` | Production with durable workflows and database managed KV. Should be used almost every time. |

---

## Basic Patterns

### Starting a project

Each project starts by importing the `durable` module and creating a new instance of the `DurableInterpreter`.
The `DurableInterpreter` is used to register and start workflows.

Workflow implementations are kept in separate files in the `workflows` folder.

```typescript
import { durable } from '@versori/run';
import { processDataWorkflow } from './workflows/durable-example';

async function main(): Promise<void> {
  const mi = await durable.DurableInterpreter.newInstance();

  // Register durable workflows
  mi.register(processDataWorkflow);

  // Start the interpreter
  await mi.start();
}

main().then().catch((err) => console.error('Failed to run main()', err));
```

### Scheduled Workflow

A scheduled workflow is a workflow that is triggered by a cron schedule. The `schedule` function takes an id, a cron schedule and an optional activation predicate. The activation predicate is used to determine which users the workflow should be run for. The `all` predicate runs the workflow for all users.

The `all` is optional and if omitted the workflow will only run for the static user.

```typescript
import { fn, schedule } from '@versori/run';

const dailySync = schedule('daily-sync', '0 0 * * *', 'all')  // Run daily at midnight
  .then(
    fn('sync-data', async (ctx) => {
      ctx.log.info('Starting sync');
      // Your logic here
      return { synced: true };
    })
  );
```

### Webhook Handler

The `webhook` trigger is to create an endpoint on the Versori platform. The endpoint can be used to send data to the workflow.

A webhook supports the following options:

- `method` - The HTTP method to use for the endpoint. Defaults to `all`.
- `cors` - Configuring CORS options on the endpoint. More details below.
- `response` - Configure the response mode `sync|async`. Defaults to `sync`.
- `connection` - The connection to use for verifying webhook signatures or authentication. Currently only support for api-key, basicauth and hmac are supported.
- `request.rawBody` - If set to `true` the raw body of the request will be passed to the context. Users will need to
handle the parsing of the body themselves, and `data` in context will be undefined. Instead,
you should call `const req = ctx.request()` to retrieve the raw request.

#### Cors config options

- `origin` - The allowed origin for the endpoint.
- `methods` - The allowed HTTP methods.
- `allowedHeaders` - The allowed request headers.
- `exposedHeaders` - The headers exposed in the response.
- `credentials` - Whether to allow credentials in CORS requests.

```typescript
import { fn, webhook } from '@versori/run';

const handleWebhook = webhook('incoming-data', {
  method: 'post',
  response: { mode: 'sync' }  // 'sync' waits for completion, 'async' returns immediately
})
  .then(
    fn('process', async (ctx) => {
      const payload = ctx.data;  // Webhook body
      ctx.log.info('Received', { payload });
      const rawRequest = ctx.request();  // Returns express.Request or undefined
      return { processed: true };
    })
  );
```

### HTTP Task with Authenticated Connection

Connections are used to authenticate requests to external systems. The `http` task takes a connection name and a callback function. The callback function is passed an `HttpContext` object that contains an authenticated `fetch` function. The `fetch` function is authenticated using the credentials for the connection.

The connections are configured outside of the code and are managed in the Versori platform. The connection name is used to link the connection to the `http` task.

```typescript
const fetchFromAPI = schedule('fetch-api', '*/5 * * * *')
  .then(
    http('get-data', { connection: 'my-api' }, async ({ fetch, log, data }) => {
      const response = await fetch('/users');  // Auth automatically added
      const users = await response.json();
      log.info('Fetched users', { count: users.length });
      return users;
    })
  );
```

### Error handling

Errors are handled using the `catch` method. The `catch` method takes a callback function that is called when an error occurs. The `catch` method can be used in combination with `then` to create a workflow that can handle errors. In the catch block the `ctx.data` will contain the thrown exception. It is preferred to not ignore errors in the execution unless it is explicitly needed.

Using try/catch inside a task is also possible but it is preferred to use the `catch` method to handle errors. Try/catch should be used to add extra logging or to do some cleanup before the workflow is ended.

```typescript
const handleErrors = schedule('fetch-api', '*/5 * * * *')
  .then(
    http('get-data', { connection: 'my-api' }, async ({ fetch, log, data }) => {
      const response = await fetch('/users');  // Auth automatically added
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

### Accessing credentials outside of an HTTP task

When using credentials outside of an HTTP task, the `ctx.credentials()` method can be used to access credentials directly. This is useful when working with authentication schemes not natively supported by Versori's HTTP task.

The `ctx.credentials()` method returns a `CredentialsProvider` object with the following methods:

| Method | Description |
|--------|-------------|
| `getRaw(name)` | Returns raw credential data as `Uint8Array`. Only applicable for API Key type credentials. |
| `getAccessToken(name, forceRefresh?)` | Returns an OAuth token object with `accessToken`, `tokenType`, and optional `expiry`. Set `forceRefresh` to `true` to force a token refresh. |
| `getOAuth1Metadata(name)` | Returns OAuth1 authorization metadata for OAuth1 authenticated systems. |

**Example:**

```typescript
webhook('example')
  .then(
    fn('use-credentials', async (ctx) => {
      // Get raw API key data
      const raw = await ctx.credentials().getRaw('my-api-key-connection');
      const apiKey = new TextDecoder('utf-8').decode(raw);

      // Get OAuth access token
      const token = await ctx.credentials().getAccessToken('my-oauth-connection');
      console.log(token.accessToken);

      // Force refresh an OAuth token
      const freshToken = await ctx.credentials().getAccessToken('my-oauth-connection', true);
    })
  );
```

The connection name (e.g., `'my-api-key-connection'`) must match a connection configured in your `versori.yaml` file.

### Creating an issue

Issues are notifications which can be alerted on in the Versori platform. The notifications are configured from the platform and can be sent to a configured email. The issues are also available in the platform for inspection.

Issues are created automatically when a `catch` block is used. The issue will contain the error message and the workflow context.

Users can also create issues manually using the `ctx.createIssue` method. This can be done from any task in the workflow. Here is how to use it:

```typescript
const issue = await ctx.createIssue({
  severity: 'low' | 'medium' | 'high',  // Required: The severity level of the issue
  title: 'Issue Title',                  // Required: A short title for the issue
  message: 'Detailed description',       // Required: A detailed message explaining the issue
  annotations: {                         // Required: Additional key-value metadata
    workflow: 'my-workflow',
    customField: 'customValue'
  }
}, {
  deduplicate: true  // Optional: If true (default), issues with the same severity and title are deduplicated
});
```

*Note*: When creating issue make sure to not create issues in a loop when deduplication is disabled. Each issue created can result in multiple emails being sent. So we need to make sure we always deduplicate issues unless we explicitly want to create a new issue.

You can use issues with custom try/catch block to report issues to the Versori platform. This should only be used if the user of the platform has explicitly asked for it.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `severity` | `'low' \| 'medium' \| 'high'` | Yes | The severity level of the issue |
| `title` | `string` | Yes | A short title describing the issue |
| `message` | `string` | Yes | A detailed message explaining what happened |
| `annotations` | `Record<string, string>` | Yes | Key-value pairs of additional metadata |

#### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `deduplicate` | `boolean` | `true` | When enabled, issues with the same severity and title are grouped together instead of creating duplicates |
| `duplicationKey` | `string` | `undefined` | A custom key to use for deduplication instead of the default (severity + title) |

#### Example: Creating an issue

```typescript
const workflow = webhook('process-data')
  .then(
    fn('process', async (ctx) => {
      // Processing logic that might fail
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
      // Create a manual issue with additional context


      return { error: ctx.data.message };
    })
  );
```

## Durable Workflows

Durable workflows are workflows that can be started programmatically from other workflows. They are designed for long-running processes and can be started from other workflows. They are queued and executed asynchronously with configurable retry policies.

### Using a Durable Workflow

Use the `workflow` trigger to use a durable workflow. The workflow can specify a `ttl` (time-to-live) in seconds, which determines how long the workflow attempt can run before timing out. The `ttl` is for this attempt and the workflow as a whole.

```typescript
import { fn, workflow } from '@versori/run';

const myWorkflow = workflow('my-workflow', { ttl: 60 })  // 60 second TTL
  .then(
    fn('process', async (ctx) => {
      // Your processing logic
      return { success: true };
    })
  );
```

### Starting a durable workflow

Durable workflows are started using either `ctx.start()` or `ctx.workflowClient().startWorkflow()` method. Both methods take the workflow id and an options object. The options object can contain the `data` (or `dataRaw` for base64 encoded data) to be passed to the workflow and the `maxAttempts` to configure the retry behavior.

It is important to note that the `startWorkflow` method returns the whole workflow object. This object can be used to wait for the workflow to complete or to check the status of the workflow.

```typescript
export const batchStartWebhook = webhook('batch-start', { response: { mode: 'sync' } })
  .then(
    fn('start-batch-workflows', async ({ data, log, openKv, workflowClient, executionId }) => {
      const wfClient = workflowClient();
      const wf = await wfClient.startWorkflow('process-data', {
        data: {},              // Data to pass to the workflow (JSON serialized)
        // dataRaw: '',        // Alternative: base64 encoded data (use data OR dataRaw, not both)
        maxAttempts: 1_000_000 // Maximum retry attempts
      });

      return {
        status: 'submitted',
        workflow: wf,
      };
    })
  )
```

#### Workflow object properties

The workflow object has the following properties:

| Property | Type | Description |
|----------|------|-------------|
| `id` | `string` | Unique identifier for the workflow |
| `projectId` | `string` | The project ID this workflow belongs to |
| `environmentId` | `string` | The environment ID this workflow belongs to |
| `group` | `string` | The workflow group name |
| `payload` | `string` | The input data (base64 encoded) |
| `output` | `string` (optional) | The output data (base64 encoded) |
| `status` | `'available' \| 'locked' \| 'completed' \| 'failed'` (optional) | Current status of the workflow |
| `attempt` | `number` | Current attempt number |
| `maxAttempts` | `number` (optional) | Maximum number of retry attempts |
| `timeout` | `string` | When the message will be delivered again |
| `createdAt` | `string` | Timestamp when the workflow was created |
| `metadata` | `WorkflowMsgMetadata` | Metadata including `executionId`, `activationId`, `userId`, and optional `parentWorkflowId` |

### Scheduling and Rescheduling Workflows

Durable workflows can reschedule themselves to process data iteratively. This is useful for batch processing where you want to process items one at a time with delays between each.

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

      // Update which item to transform next
      index.current++;
      await kv.set(`${executionId}/index`, index);
      log.info('Updated index', index);

      // Schedule next item or complete
      const wfClient = workflowClient();
      if (index.current < index.total) {
        log.info('Rescheduling workflow', { workflow });
        await wfClient.rescheduleWorkflow(workflow, '5s');  // Reschedule after 5 seconds
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

  // The workflow will automatically retry based on maxAttempts
  throw error;
});
```

### WorkflowClient Methods

The `workflowClient()` provides methods for controlling workflow execution:

| Method | Description |
|--------|-------------|
| `startWorkflow(group, opts)` | Starts a new workflow with the given group/id and options |
| `getWorkflow(workflow)` | Gets the current state of a workflow |
| `getWorkflowById(id)` | Gets a workflow by its ID |
| `waitForCompletion(workflow)` | Waits for a workflow to complete (polls until status is 'completed' or 'failed') |
| `rescheduleWorkflow(workflow, delay)` | Reschedules the workflow to run again after the specified delay (e.g., `'5s'`, `'1m'`) |
| `completeWorkflow(workflow, status, result)` | Marks the workflow as complete with a status ('completed' or 'failed') and result object |

### Using KV Storage with Durable Workflows

Durable workflows commonly use the key-value storage to track progress across reschedules. Use the `executionId` to namespace your data:

```typescript
const kv = await openKv();

// Store progress using executionId as namespace
await kv.set(`${executionId}/index`, { current: 0, total: items.length });
await kv.set(`${executionId}/items/0`, items[0]);

// Retrieve progress
const index = await kv.get(`${executionId}/index`);
```

## Context Object (`ctx`)

The context is passed to every task and provides:

```typescript
interface Context<D> {
  data: D;                    // Input data from previous task
  log: Logger;                // Structured logging
  executionId: string;        // Unique execution ID
  startTime: Date;            // When execution started
  activation: Activation;     // Current activation (user context)
  workflow?: Workflow;        // Workflow object (durable workflows only)

  // Methods
  openKv(scope?): KeyValue;           // Key-value storage
  credentials(): CredentialsProvider;  // Access credentials outside of the managed http connections.
  createIssue(issue, options?): Promise<Issue | null>;  // Create issues/alerts
  start(workflowId, opts): Promise<AsyncWorkflow>;  // Start durable workflow
  request(): express.Request | undefined;          // Raw HTTP request (webhooks only)
  workflowClient(): WorkflowInterface; // Workflow client to start, get and reschedule workflows.
  destroy(scope): Promise<void>;      // Destroy KV store for a given scope
}
```

### Activation on the context

---

The `ctx.activation` contains information about the current activation. An activation represents a user's activation of an integration, containing user-specific configuration and credentials for accessing external systems.

```typescript
interface Activation {
  id: string;
  user: EndUser;
  connections?: Array<Connection>;
  dynamicVariables?: DynamicVariables;
  getVariable<T = unknown>(name: string): T | undefined;
  setVariable(name: string, value: unknown): Promise<void>;
}
```

#### Properties

| Property | Type | Description |
| -------- | ---- | ----------- |
| `id` | `string` | The unique identifier for this activation |
| `user` | `EndUser` | The end user who owns this activation |
| `connections` | `Array<Connection>` | The connections available for the current user |
| `dynamicVariables` | `DynamicVariables` | Key-value store of dynamic variables for this activation |

#### Methods

**`getVariable<T>(name: string): T | undefined`**

Retrieves a dynamic variable value for this activation. Returns `undefined` if no dynamic variables exist or the variable is not found.

```typescript
const lastSyncDate = ctx.activation.getVariable<string>('lastSyncDate');
const settings = ctx.activation.getVariable<boolean>('settings');
```

**`setVariable(name: string, value: unknown): Promise<void>`**

Sets a dynamic variable value for this activation. This persists the value to the platform and updates the local cache.

```typescript
await ctx.activation.setVariable('lastSyncDate', new Date().toISOString());
await ctx.activation.setVariable('settings', true);
```

#### Related Types

```typescript
interface EndUser {
  id: string;           // Versori identifier for the user
  externalId: string;   // Your system's identifier for the user
  displayName: string;  // Human-readable name
  organisationId: string;
  createdAt: string;
  updatedAt: string;
}

type DynamicVariables = {
  [key: string]: unknown;
};
```

## Key-Value Storage

The key-value storage is used to store data across executions. The key-value storage is scoped to the execution, project, or organization.

Below are the core functions of the key-value storage.

```typescript
const kv = ctx.openKv(':project:');  // ':execution:', ':project:' or ':organization:'

await kv.set(['users', id], userData);
const user = await kv.get(['users', id]);
await kv.delete(['users', id]);
const list = await kv.list(['users']);
const count = await kv.count(['users']);
```

## Common Patterns for Agents

### Data Transformation Pipeline

```typescript
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

### Error Handling

```typescript
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

### Re-try functionality

There is no built-in retries outside of durable workflows. So if an error occurs in a non-durable workflow it is up to the workflow to handle the error and decide if it should be retried.

Most commonly you would want to implement an exponential backoff retry strategy.

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
      
      // Exponential backoff: 2^attempt * 100ms (100ms, 200ms, 400ms, 800ms, ...)
      const delayMs = Math.pow(2, attempt) * 100;
      await new Promise((resolve) => setTimeout(resolve, delayMs));
    }
  }
  
  throw lastError;
}
```

#### HTTP request retry example

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
    // Combine timeout with optional external signal
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

      // If externally aborted, don't retry
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

// Usage within an http task with ctx.log
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

## HttpContext (for `http` tasks)

The `http` task provides an extended context:

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
  method: 'post' | 'get' | 'put' | 'delete' | 'patch' | 'all',
  cors: true | { origin, methods, allowedHeaders, credentials },
  response: {
    mode: 'sync' | 'async',
    onSuccess: (ctx) => Response,
    onError: (ctx) => Response
  },
  connection: 'connection-name',  // For webhook signature verification
  request: { rawBody: true }      // Pass raw body instead of parsed
});
```

---

## Quick Reference Cheat Sheet

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

## Type Signatures for Reference

```typescript
// Task creation
function fn<In, Out>(id: string, callback: (ctx: Context<In>) => Out | Promise<Out>): Task<In, Out>;
function http<In, Out, PageParams>(id: string, opts: HttpOptions<PageParams>, callback: (ctx: HttpContext<In, PageParams>) => Promise<Out>): Task<In, Out>;

// Triggers
function schedule(id: string, cron: string, activationPredicate?: ActivationPredicate): Workflow<ScheduleData>;
function webhook(id: string, options?: WebhookOptions): Trigger<WebhookData>;
function workflow(id: string, options?: DurableWorkflowOptions): Trigger<DurableWorkflowData>;

// Key-Value
interface KeyValue {
  get<T>(key: string | string[], options?: GetOptions<T>): Promise<T | undefined>;
  set<T>(key: string | string[], value: T): Promise<void>;
  delete(key: string | string[]): Promise<void>;
  list(prefix: string[], options?: ListKVRequest): Promise<ListKVResponse>;
  count(prefix: string[]): Promise<CountKVResponse>;
}
```

---

## Best Practices for Generating Code

1. **Always give tasks meaningful IDs** - They appear in execution traces
2. **Use structured logging** - `ctx.log.info('action', { key: value })`
3. **Use `ctx.log` for logging** - This provides structured logging and observability. NEVER template the log message. Instead use a static message and pass the dynamic data in the structured arguments. This makes searching logs easier. NEVER use emoji's or colors in log messages.
4. **Handle errors with `.catch()`** - This improves observability when a workflow fails.
5. **Return data from tasks** - Next task receives it via `ctx.data`
6. **Use `http` for API calls** - Gets automatic auth from connections
7. **Use `fn` for data processing** - Pure logic without external calls
