# versori-skills

A collection of Skills for working with the [Versori](https://versori.com) platform. Skills are an open standard for packaging AI agent knowledge and workflows, usable across AI coding tools such as Claude Code and Cursor.

## Skills

### `coding-versori-sdk`

Helps Claude write expert-level data integration code using the [versori-run](https://www.npmjs.com/package/@versori/run) SDK. Triggers when the user wants to build or debug ETL processes, API integrations, data transformation pipelines, webhooks, database sync workflows, or any other data integration service.

**Covers:**

- TypeScript workflow authoring using the versori-run SDK
- Schedule, webhook, and durable workflow patterns
- KV store usage for incremental sync and stateful batch processing
- API research and documentation before code generation
- Versori CLI usage — creating projects, switching contexts, deploying

## Structure

```
coding-versori-sdk/
├── SKILL.md                  # Core workflow patterns and critical rules
└── references/
    ├── research-docs.md      # Research document structure and guidelines
    ├── sdk-guide.md          # Comprehensive guide on how to use the versori/run SDK for implementing an integration
    └── cli-usage.md          # CLI tools, environment variables, deployment safety
```
