# Versori CLI Documentation

Welcome to the official documentation for the Versori CLI. This guide will help you configure the CLI, understand its core concepts, and get started with managing your Versori projects.

## Quickstart

1. **Install the CLI**: Follow the installation instructions in the [main README](../README.md).
2. **Add a Context**: Configure your authentication and organisation details by adding a context.

   ```sh
   versori context add
   ```

3. **Sync a Project**: Download your project files to a local directory.

   ```sh
   versori projects sync --project <project-id>
   ```

   *Note: The `--project` flag is only required the first time you sync a project into a directory.*

## Configuring the CLI

The Versori CLI uses a configuration file to manage your settings and authentication profiles.

- **Default Location**: `~/.versori/config.yaml`
- **Environment Variable**: You can override the default location by setting the `VERSORI_CONFIG` environment variable.
- **Global Flags**:
  - `--config` or `-c`: Specify a custom path to the config file.
  - `--context` or `-x`: Temporarily override the active context for a single command.
  - `--output` or `-o`: Change the output format of commands (`table`, `json`, `yaml`).

## What is a Context?

A **Context** is a named configuration profile that stores your authentication and organisation details. It allows you to easily switch between different organisations, or user accounts without having to re-authenticate.

A context contains:

- **Name**: A unique identifier for the profile.
- **Organisation ID**: The Versori organisation you are interacting with.
- **JWT**: Your authentication token.

You can manage your contexts using the `versori context` commands:

- `versori context add`: Add a new context.
- `versori context select`: Switch your active context.

## How the `.versori` File Works

When you run `versori projects sync` to download a project, the CLI automatically generates a `.versori` file in the root of the sync directory.

### Features of the `.versori` file

1. **Automatic Project ID Resolution**: It stores the `ProjectId`. When you run subsequent commands inside this directory, the CLI automatically reads the `.versori` file to infer the project ID. You no longer need to pass the `--project` flag.
2. **Context Safety**: It stores the name of the `Context` used to sync the project. The CLI will verify that your currently active context matches the one in the `.versori` file before executing commands. This prevents you from accidentally modifying a project using the wrong credentials or organisation.
3. **Flag Validation**: If you explicitly pass the `--project` flag while inside a synced directory, the CLI will check it against the `.versori` file. If they do not match, the command will fail, ensuring you don't accidentally run commands against a different project while in another project's working directory.
