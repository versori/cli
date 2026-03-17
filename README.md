# Versori CLI

A command-line interface for managing resources on the [Versori](https://versori.com) platform — projects, systems, connections, users, and more.

## Documentation

For comprehensive documentation on all commands and features, please visit the [docs](./docs) directory.

You can also read the [Skills Documentation](./skills/README.md) to learn about our collection of Agent Skills for working with the Versori platform across AI coding tools.

## Installation

### Install script (recommended)

The quickest way to install the Versori CLI is with the install script. It automatically detects your OS and architecture, downloads the latest release binary from GitHub, and places it in `/usr/local/bin`.

```sh
curl -fsSL https://raw.githubusercontent.com/versori/cli/main/install.sh | sh
```

To pin to a specific version, set `VERSORI_VERSION`:

```sh
VERSORI_VERSION=v0.0.1 curl -fsSL https://raw.githubusercontent.com/versori/cli/main/install.sh | sh
```

Supported platforms: Linux and macOS, on `amd64` and `arm64`. Windows users should download a pre-built binary from the releases page instead.

### Download a pre-built binary

Pre-built binaries for Linux, macOS, and Windows (amd64 and arm64) are available on the [GitHub releases page](https://github.com/versori/cli/releases). Download the archive for your platform, extract it, and move the `versori` binary somewhere on your `PATH`.

### Build from source

#### Prerequisites

- [Go](https://go.dev/dl/) 1.26+

```sh
make cli
```

This builds the `versori` binary and copies it to `$GOPATH/bin/`. Make sure that directory is in your `PATH`.

Alternatively, build without installing:

```sh
make build
```

The binary will be placed in `./bin/`.

## Using Skills in AI Coding Tools

This repo includes AI agent skills that help your AI coding tool write expert-level data integration code using the [versori-run](https://www.npmjs.com/package/@versori/run) SDK. The skills trigger when you ask your AI coding tool to build or debug ETL processes, API integrations, data transformation pipelines, webhooks, or any other data integration service.

### Claude Code

**Via the marketplace:**

1. Add the Versori marketplace:
   ```
   /plugin marketplace add versori/cli
   ```
2. Install the skill:
   ```
   /plugin install versori-skills@versori-cli
   ```

You can also install from the terminal outside of Claude Code:
```sh
claude plugin install versori-skills@versori-cli
```

**Via the Versori CLI:**

```sh
versori skills download
```

### Other AI Coding Tools

Use the Versori CLI to extract skills locally into your project:

```sh
versori skills download
```

To install skills globally (e.g. for Cursor):

```sh
versori skills download --directory ~/.cursor/skills/
```

> Use the `--agent` flag (e.g. `versori skills download --agent`) to combine all skills into a single `AGENTS.md` file.
