# Practical usage

## Inspect the computed environment

When no command is provided, `we` runs `env`. This is useful while building configuration:

```bash
we -e env/dev.yml
we --clean -e env/dev.yml
```

Use `--clean` to see only variables that withenv loaded.

## Layer configuration by context

A common layout is:

```text
env/
  base.yml
  dev.yml
  prod.yml
```

Then run:

```bash
we -e env/base.yml -e env/dev.yml ./my-app
we -e env/base.yml -e env/prod.yml ./my-app
```

Later files override earlier files, so each environment file only needs to contain the values that differ.

## Keep aliases in the repository

For commands that need several inputs, use an alias file:

```yaml title="env/dev.alias.yml"
---
- file: base.yml
- file: dev.yml
- envvar: APP_ENV=development
```

Run it with:

```bash
we -a env/dev.alias.yml ./my-app
```

If the alias is named `.withenv.yml` in your project tree, `we` loads it automatically for commands run below that directory:

```bash
we ./my-app
```

## Load secrets dynamically

Use `--script` when another tool can print JSON or YAML:

```bash
we -e env/base.yml --script 'vault kv get -format=json secret/my-app | jq .data.data' ./my-app
```

Use backtick command substitution when a single variable depends on a command:

```yaml
---
- VAULT_TOKEN: "`vault write -format=json auth/approle/login role_id=$ROLE_ID secret_id=$SECRET_ID | jq -r .auth.client_token`"
```

## Render config files

Some tools require config files instead of environment variables. Render a file before the command starts:

```bash
we -e env/prod.yml -t config/app.ini.tmpl ./my-app --config config/app.ini
we -e env/prod.yml -t 'config/app.ini.tmpl:/etc/my-app/app.ini' ./my-app
```

The first form writes to `config/app.ini` because `.tmpl` is removed.

## Use .envrc safely

`we` can read simple `.envrc` files automatically:

```bash
export APP_ENV=development
DATABASE_URL=postgres://localhost/myapp
```

It parses assignments but does not execute arbitrary shell code. Disable this for one command with:

```bash
we --no-direnv -e env/dev.yml ./my-app
```

## Convert dotenv files

Bring an existing `.env` file into withenv:

```bash
we convert .env -o env/local.yml
```

## Run agent tools with protections

For AI agent workflows, use `--agent`:

```bash
we --agent -e env/dev.yml claude
```

Add project-specific sandbox rules in `.withenv.yml`:

```yaml
---
- file: env/dev.yml
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

See [Reference](reference.md) for exact semantics of every flag and file format.
