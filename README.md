# Withenv (`we`)

Withenv loads environment variables from YAML, JSON, scripts, directories, and
aliases before running a command. It makes runtime configuration explicit and
repeatable without permanently changing your shell.

## Install

```bash
brew install ionrock/tap/we
# or
go install github.com/ionrock/we/cmd/we@latest
```

## Quickstart

Create `dev.yml`:

```yaml
---
APP_ENV: development
LOG_LEVEL: debug
DATABASE_URL: postgres://localhost/myapp
```

Run a command with those variables:

```bash
we --env dev.yml printenv APP_ENV
# development
```

Layer files and override values from left to right:

```bash
we -e env/base.yml -e env/dev.yml -E PORT=9000 ./my-app
```

Inspect exactly what withenv loads:

```bash
we --clean -e dev.yml
```

Create a reusable `.withenv.yml` alias in your project root:

```yaml
---
- file: env/base.yml
- file: env/dev.yml
- envvar: APP_ENV=development
```

Then run any command below that directory with the alias loaded automatically:

```bash
we ./my-app
```

Convert an existing dotenv-style file:

```bash
we convert .env --output env/local.yml
we -e env/local.yml ./my-app
```

Render config files before launching a command:

```bash
we -e dev.yml --template app.conf.tmpl ./my-app --config app.conf
```

## Common flags

- `-e, --env FILE`: load YAML/JSON environment file (repeatable)
- `-d, --directory DIR`: recursively load YAML/JSON files (repeatable)
- `-a, --alias FILE`: load a withenv alias file (repeatable)
- `-E, --envvar KEY=VALUE`: set one variable (repeatable)
- `-s, --script COMMAND`: load YAML/JSON from command stdout (repeatable)
- `-t, --template TEMPLATE[:TARGET]`: render a Go template before running
- `-c, --clean`: run with only variables loaded by withenv sources
- `--no-direnv`: disable automatic `.envrc` loading
- `--agent`: run with AI-agent sandbox protections

Run `we --help` and `we convert --help` for the full CLI reference.

## Documentation

The full docs include:

- [Quickstart](docs/quickstart.md)
- [Practical usage](docs/usage.md)
- [Examples](docs/examples.md)
- [Reference](docs/reference.md)

Published docs: <https://withenv.readthedocs.org>
