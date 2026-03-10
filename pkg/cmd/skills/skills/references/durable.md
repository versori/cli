# Durable Workflows Reference

Use `DurableInterpreter` for long-running processes, batch processing with retry logic, or guaranteed execution. **Never mix `MemoryInterpreter` and `DurableInterpreter`** in the same application.

## Interpreter Choice

| Interpreter | When to use |
|---|---|
| `DurableInterpreter` | **Default.** Production workflows with persistent KV. Use almost every time. |
| `MemoryInterpreter` | Development, testing, simple deployments without persistent KV. |

## Structure

```typescript
import { durable, workflow, http, fn } from '@versori/run';

const durableWorkflow = durable(
    'process-large-batch',
    {
        group: 'batch-processing',
        maxAttempts: 5,         // default: 10
    },
    workflow()
        .then(
            http('fetch-batch', { connection: 'source' }, async ({ fetch, activation }) => {
                const batchId = activation.metadata.get('batchId');
                const response = await fetch(`/batches/${batchId}`);
                return await response.json();
            })
        )
        .then(
            fn('process-items', ({ data }) => {
                return data.items.map(item => ({
                    ...item,
                    processed: true,
                    timestamp: new Date().toISOString()
                }));
            })
        )
);

async function main(): Promise<void> {
    const di = await durable.DurableInterpreter.newInstance();
    di.register(durableWorkflow);
    await di.start();

    // Schedule a durable workflow execution
    await di.schedule(durableWorkflow, {
        payload: { batchSize: 100 },
        metadata: new Map([
            ['executionId', crypto.randomUUID()],  // executionId is mandatory
            ['batchId', 'batch-123']
        ])
    });
}

main().catch((err) => console.error('Failed to run main()', err));
```

## Options

| Field | Description |
|---|---|
| `group` | Workflow organization label |
| `maxAttempts` | Max retries before failing (default: 10) |
| `metadata` | `Map<string, string>` — `executionId` is mandatory |
| `payload` | User-submitted input data (stored as bytes) |
| `attempt` | Available in context — tracks current retry count |
