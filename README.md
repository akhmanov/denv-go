# denv

A simple CLI utility for Go to manage environment variables from `.env` files.

It allows you to load environment variables from one or more `.env` files and execute commands, or inspect the loaded environment.

## Installation

```bash
go install github.com/akhmanov/denv-go/cmd/denv@latest
```

## Usage

### Execute a command

Run a command with environment variables loaded from `.env` files.

```bash
denv exec -- printenv PORT
```

By default, `denv` looks for a `.env` file in the current directory.

### Specify multiple files

You can load multiple files. Values from later files override earlier ones.

```bash
denv -f .env -f .env.local exec -- ./server
```

### Inspect environment

#### Get a specific value

```bash
denv get PORT
```

#### List all keys

```bash
denv keys
```

#### Dump all variables

```bash
denv list
```

## Behavior

1. **System Environment**: `denv` starts with the current system environment (`os.Environ()`).
2. **Overrides**: It loads `.env` files in the order specified. Variables defined in these files override system environment variables and variables from previous files.
3. **Exit Codes**: The `exec` command propagates the exit code of the executed command.
4. **Signals**: `exec` forwards system signals (SIGINT, SIGTERM, etc.) to the child process.

## License

MIT
