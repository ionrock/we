# Advanced usage

## Agent mode

`--agent` runs a command with protections intended for AI coding agents.

```bash
we --agent claude
we --agent -e devenv.yml claude
```

Agent mode does two things:

1. Wraps the command in the platform sandbox implementation.
2. Writes temporary Claude Code hooks so Bash tool commands are prefixed with `we` and re-evaluate the environment.

This is useful when an agent needs project environment variables but should not freely read local credentials.

In agent mode, withenv source files are also denied to the child command after they are loaded. This includes:

- `.withenv.yml` aliases and the files/directories they reference
- Explicit `--env` YAML/JSON files
- Explicit `--directory` environment directories
- `.envrc` files
- `.env` / `source_env` files loaded from `.envrc`
- `~/.withenv_global.yml`, when present

The agent receives the computed environment, but cannot read the source files through the sandbox.

### Default denied paths

The sandbox denies common sensitive paths when they exist, including:

- `~/.ssh`
- `~/.aws`
- `~/.gnupg`
- `~/.config/gcloud`
- `~/.azure`
- `~/.config/op`
- `~/.netrc`
- `~/.npmrc`
- `~/.pypirc`
- `~/.docker/config.json`

### Add sandbox rules with flags

```bash
we --agent \
  --sandbox-deny ~/.kube \
  --sandbox-allow ~/.kube/cache \
  --sandbox-deny-network \
  claude
```

### Add sandbox rules in `.withenv.yml`

```yaml title=".withenv.yml"
---
- file: devenv.yml
- sandbox:
    deny:
      - ~/.kube
      - ~/.config/gh
    allow:
      - ~/.config/gh/hosts.yml
    deny-network: true
    skip-prefix:
      - git
```

`skip-prefix` lists commands that Claude Code hooks should not rewrite with a leading `we`.

## Scripts

Use `--script` when a command can print YAML or JSON for withenv to load.

```bash
we --script './print-env-json' ./my-app
```

Example script output:

```json
{
  "APP_ENV": "development",
  "LOG_LEVEL": "debug"
}
```

A common pattern is loading secrets from another tool:

```bash
we -e devenv.yml \
  --script 'vault kv get -format=json secret/my-app | jq .data.data' \
  ./my-app
```

Scripts are repeatable and are applied in order with other flags:

```bash
we -e base.yml --script './secrets' -E LOG_LEVEL=trace ./my-app
```

## Command substitution in values

A string value that starts and ends with a backtick is executed as a command. The command stdout becomes the value.

```yaml title="devenv.yml"
---
- ROLE_ID: my-role
- VAULT_TOKEN: "`vault write -format=json auth/approle/login role_id=$ROLE_ID secret_id=$SECRET_ID | jq -r .auth.client_token`"
```

Command values are split lexically and support simple pipelines with `|`; they are not evaluated by a shell.

Use this for single values that must be computed at runtime. Prefer `--script` when an external command naturally returns several variables.

## Templates

Use `--template` when an application needs a config file instead of environment variables.

Create an environment file:

```yaml title="devenv.yml"
---
APP_ENV: development
DATABASE_URL: postgres://localhost/myapp
PORT: "8080"
```

Create a template:

```text title="app.conf.tmpl"
env={{ .APP_ENV }}
database={{ .DATABASE_URL }}
port={{ .PORT }}
```

Render the template before running your app:

```bash
we -e devenv.yml --template app.conf.tmpl ./my-app --config app.conf
```

Because no target was provided, `app.conf.tmpl` is written to `app.conf`.

To write to an explicit target, use `TEMPLATE:TARGET`:

```bash
we -e devenv.yml --template 'app.conf.tmpl:/etc/my-app/app.conf' ./my-app
```

Templates use Go's `text/template` package plus Sprig functions. The template data is the environment map, so these are both valid:

```text
port={{ .PORT }}
port={{ env "PORT" }}
```

## Directory-based environments

For larger projects, group files under a directory and load them recursively:

```text
env/dev/
  app.yml
  database.yml
  services.yml
```

```bash
we --directory env/dev ./my-app
```

This is useful when different teams or subsystems own different parts of the environment.

## Dotenv and direnv-style `.envrc`

`we` follows direnv's model: it auto-loads `.envrc`, and `.envrc` can opt into loading dotenv files.

```bash title=".envrc"
dotenv
```

This loads `.env` next to the `.envrc`. You can also provide a path:

```bash title=".envrc"
dotenv .env.local
dotenv_if_exists .env.secrets
```

For shared assignment files, use:

```bash title=".envrc"
source_env env/common.env
source_env_if_exists env/local.env
```

`we` parses assignments from those files but does not execute arbitrary shell code.

## Converting dotenv files

If you prefer withenv YAML, convert existing `.env` files:

```bash
we convert .env --output devenv.yml
```

The converter supports simple assignment lines:

```bash
APP_ENV=development
export LOG_LEVEL=debug
```

It does not execute shell code.
