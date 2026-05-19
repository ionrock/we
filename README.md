# Withenv (`we`)

Withenv loads environment variables from YAML, JSON, scripts, directories, and aliases before running a command. It makes runtime configuration explicit and repeatable without permanently changing your shell.

## Install

```bash
brew install ionrock/tap/we
# or
go install github.com/ionrock/we/cmd/we@latest
```

## Quickstart

Create `devenv.yml`:

```yaml
---
APP_ENV: development
LOG_LEVEL: debug
DATABASE_URL: postgres://localhost/myapp
```

Create `.withenv.yml`:

```yaml
---
- file: devenv.yml
```

Prefix commands with `we`:

```bash
we printenv APP_ENV
# development

we ./my-app
we go test ./...
```

Inspect exactly what withenv loads:

```bash
we --clean
```

## Documentation

The docs are organized into three sections:

- [Quickstart](docs/quickstart.md): set up `.withenv.yml` and `devenv.yml`.
- [Reference](docs/reference.md): every CLI flag and option.
- [Advanced usage](docs/advanced.md): `--agent`, scripts, and templates.

Published docs: <https://withenv.readthedocs.org>
