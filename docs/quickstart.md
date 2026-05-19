# Quickstart

This quickstart sets up the smallest useful withenv project: a `.withenv.yml` alias file and a `devenv.yml` file containing environment variables. After that, prefix commands with `we` and they run with your project environment.

## Install

```bash
brew install ionrock/tap/we
# or
go install github.com/ionrock/we/cmd/we@latest
```

Verify the CLI is available:

```bash
we --version
```

## Create `devenv.yml`

Create a file named `devenv.yml` in your project:

```yaml title="devenv.yml"
---
APP_ENV: development
LOG_LEVEL: debug
DATABASE_URL: postgres://localhost/myapp
PORT: "8080"
```

These values will be available to commands launched through `we`.

## Create `.withenv.yml`

Create a `.withenv.yml` alias file next to `devenv.yml`:

```yaml title=".withenv.yml"
---
- file: devenv.yml
```

When you run `we` anywhere in this directory tree, it searches upward for `.withenv.yml` and loads it automatically.

## Prefix commands with `we`

Run commands exactly as you normally would, but prefix them with `we`:

```bash
we printenv APP_ENV
# development

we go test ./...
we npm run dev
we ./my-app
```

Your current shell is unchanged. The variables are applied only to the child command.

## Inspect what will be loaded

With no command, `we` runs `env`:

```bash
we
```

To see only variables loaded by withenv sources, use `--clean`:

```bash
we --clean
```

Add `--debug` / `-D` when you want to see how `we` discovers and applies files:

```bash
we --debug --clean
we -D --clean printenv APP_ENV
```

## Override one value for a command

Use `--envvar` / `-E` for one-off overrides:

```bash
we -E LOG_LEVEL=trace ./my-app
```

CLI flags are applied after `.withenv.yml`, so the override wins.

## Optional: load an existing `.env` through `.envrc`

Like direnv, `we` auto-loads `.envrc`, not `.env` directly. If you already have a dotenv-style `.env`, create this `.envrc`:

```bash title=".envrc"
dotenv
```

That loads `.env` from the same directory. You can also name a file explicitly:

```bash title=".envrc"
dotenv .env.local
```

## Add another environment file

You can layer files by adding entries to `.withenv.yml`:

```yaml title=".withenv.yml"
---
- file: devenv.yml
- file: local.yml
```

Later entries override earlier entries. This is useful for local machine-specific values that should not be shared.

## Next steps

- Use [Reference](reference.md) for every CLI flag and option.
- Use [Advanced usage](advanced.md) for scripts, templates, and `--agent`.
