# Research Documents Reference

Research documents capture API findings BEFORE writing any integration code. Use any available search or web fetch tools to retrieve up-to-date API documentation directly from official sources. By documenting endpoints, schemas, authentication, and integration patterns upfront, the generated workflow code is correct on the first attempt rather than relying on guesswork.

## When to Research

Create a research document by default for every new integration. Skip only when:
- The user provides complete API documentation (endpoints, schemas, auth details)
- You are fully confident in the endpoint details for well-known APIs (e.g. Stripe, Shopify)

## Document Structure

Write the research document as `research.md` in a `versori-research/` directory in the project root. Include these sections:

### System & Authentication

- Source and target system names
- API type (REST, GraphQL, SOAP)
- Authentication type needed (API key, OAuth2, basic auth)
- Any end-system setup required (enabling API access, creating apps, scopes)
- Base URLs and API versions
- User-specific configuration required for each system (e.g., shop domain, subdomain, instance URL, tenant ID) — note any values that the user must provide so they can be passed via the `--system-overrides` flag at bootstrap time

### API Endpoints

For each endpoint needed by the integration:
- URL path and HTTP method
- Path and query parameters
- Request body JSON schema (with required vs optional fields)
- Response body JSON schema
- Status codes and their meanings

### Integration Patterns

- Rate limits and throttling behaviour
- Pagination approach (cursor, offset, link headers)
- Error codes and retry strategies
- Webhook event types and payload schemas (if applicable)

### Data & Transformations

- Field mappings between source and target systems
- Data type conversions (dates, enums, IDs)
- Validation rules and constraints
- Sync strategy (full sync, incremental, delta)

### Source Attribution

Every fact must cite its source URL inline. Link directly to the documentation page where the information was found.

## Excluded Content

Versori handles authentication, deployment, and runtime concerns automatically. Do not include:

- **Document metadata** — no version numbers, dates, "prepared by" headers
- **Code examples or snippets** — the research doc captures facts, not implementation
- **Token management** — no token acquisition, refresh, storage, or expiry logic
- **OAuth flow implementation** — no authorization endpoints, redirects, or PKCE details
- **Secret or credential storage** — Versori manages credentials via connections
- **Deployment or hosting** — no Docker, SDK installation, or infrastructure setup

## Scope

Only document the endpoints and patterns actually needed for the integration. Do not document the entire API surface — focus on what the workflow will use.

## Updating

If the user asks to expand the integration (e.g. adding new endpoints or a new system), update the existing `versori-research/research.md` rather than creating a new file.

## Next Step

The research document serves as input for `versori projects systems bootstrap`, which reads the System & Authentication section to create the corresponding systems in the project. Ensure all system names are accurately listed in that section, as bootstrap uses these to create systems. If any system requires user-specific configuration, those values are passed via the `--system-overrides` flag on the bootstrap command rather than being embedded in this document. After bootstrapping, verify the created systems with `versori projects systems list`, then create connections for each system using `versori connections create` with `--bypass` before proceeding to code generation.
