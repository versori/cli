# KV Store Reference

Use the KV store **only** for data that persists between executions AND is both SET and READ within the workflow.

## API

```typescript
export interface KeyValue {
    get<T = any>(key: string, options?: GetOptions<T>): Promise<T | undefined>;
    set<T>(key: string, value: T): Promise<void>;
    delete(key: string): Promise<void>;
}
```

- `get('key')` → `Promise<T | undefined>`
- `get('key', { defaultValue: 0 })` → `Promise<T>` (type inferred from default)
- `get('key', { throwIfNotFound: true })` → `Promise<T>`

## Correct Usage

**Incremental sync** — read last sync time, use it, update it:
```typescript
http('fetch-data', { connection: 'shopify' }, async ({ fetch, openKv }) => {
    const kv = openKv();
    const lastSync = await kv.get<string>('lastSync');

    const params = new URLSearchParams();
    if (lastSync) params.append('updated_at_min', lastSync);

    const response = await fetch(`/admin/api/2023-10/products.json?${params}`);
    await kv.set('lastSync', new Date().toISOString());
    return await response.json();
})
```

**Batch pagination** — track current page across executions:
```typescript
const currentPage = await kv.get<number>('currentPage', { defaultValue: 1 });
const data = await fetch(`/api/data?page=${currentPage}`).then(r => r.json());
await kv.set('currentPage', data.hasMore ? currentPage + 1 : 1);
```

## Incorrect Usage — Do Not Do This

```typescript
// ❌ Write-only — value is never read back
await kv.set('last_sync_time', newTime);

// ❌ Static config — use integration variables instead
await kv.set('api_version', '2023-10');

// ❌ Logging — use log.info/log.error instead
await kv.set('execution_status', 'completed');
```

## Alternatives

| Need | Use instead |
|---|---|
| User-configurable settings | `activation.getVariable()` / `activation.setVariable()` |
| Data used only in current execution | Local variables |
| Debugging/monitoring | `log.info`, `log.error` |
| Hardcoded constants | TypeScript `const` |
