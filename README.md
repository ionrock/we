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

## More complete example

Use a YAML list when ordering matters. Later entries can reference values from earlier entries, and later files or flags override earlier values.

```yaml
# devenv.yml
---
- APP_ENV: development
- APP_HOST: localhost
- APP_PORT: "8080"
- BASE_URL: http://$APP_HOST:$APP_PORT
- LOG_LEVEL: debug
```

```yaml
# local.yml, optional and usually gitignored
---
- APP_PORT: "9000"
- LOG_LEVEL: trace
```

```yaml
# .withenv.yml
---
- file: devenv.yml
- file: local.yml
```

Because sources are applied in order, `local.yml` overrides `devenv.yml`:

```bash
we --clean printenv APP_PORT
# 9000
```

Explicit flags are applied after `.withenv.yml`, from left to right, so the final flag wins:

```bash
we -E APP_PORT=7000 -E APP_PORT=7100 --clean printenv APP_PORT
# 7100
```

You can also render a simple config file before the command starts:

```text
# app.conf.tmpl
env={{ .APP_ENV }}
url={{ .BASE_URL }}
log_level={{ .LOG_LEVEL }}
```

```bash
we --template app.conf.tmpl cat app.conf
# env=development
# url=http://localhost:8080
# log_level=trace
```

By default, `app.conf.tmpl` renders to `app.conf`. Use `TEMPLATE:TARGET` to choose an explicit output path.

## Documentation

The docs are organized into three sections:

- [Quickstart](docs/quickstart.md): set up `.withenv.yml` and `devenv.yml`.
- [Reference](docs/reference.md): every CLI flag and option.
- [Advanced usage](docs/advanced.md): `--agent`, scripts, and templates.

Published docs: <https://withenv.readthedocs.org>
