# Versori CLI

A command-line interface for managing resources on the [Versori](https://versori.com) platform — projects, systems, connections, users, and more.

## Documentation

For comprehensive documentation on all commands and features, please visit the [docs](./docs) directory.

You can also read the [Skills Documentation](./pkg/cmd/skills/skills/README.md) to learn about our collection of Agent Skills for working with the Versori platform across AI coding tools.

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
