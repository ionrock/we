# Quickstart

## Install `we`

Using Homebrew:

```bash
brew install ionrock/tap/we
```

Or with Go:

```bash
go install github.com/ionrock/we/cmd/we@latest
```

## Create your first environment

Create `dev.yml`:

```yaml
---
APP_ENV: development
LOG_LEVEL: debug
DATABASE_URL: postgres://localhost/myapp
```

Run a command with those values:

```bash
we --env dev.yml printenv APP_ENV
# development
```

Short flags work too:

```bash
we -e dev.yml env | grep LOG_LEVEL
```

## Layer values

Sources are applied from left to right, so later values win:

```yaml title="base.yml"
---
LOG_LEVEL: info
PORT: "8080"
```

```yaml title="local.yml"
---
LOG_LEVEL: debug
```

```bash
we -e base.yml -e local.yml printenv LOG_LEVEL
# debug
```

## Set one-off overrides

Use `--envvar` / `-E` for values you do not want to put in a file:

```bash
we -e dev.yml -E PORT=9000 printenv PORT
```

## Use a directory

Put multiple YAML or JSON files under a directory and load them recursively:

```bash
we --directory envs/dev ./my-app
we -d envs/dev ./my-app
```

## Create a reusable alias

Create `.withenv.yml` in your project root:

```yaml
---
- file: env/base.yml
- file: env/dev.yml
- envvar: APP_ENV=development
```

Now any command under that directory automatically loads the alias before explicit flags:

```bash
we ./my-app
```

You can also name an alias file explicitly:

```bash
we --alias env/dev.alias.yml ./my-app
```

## Import an existing .env file

Convert a dotenv-style file to withenv YAML:

```bash
we convert .env --output env/local.yml
we -e env/local.yml ./my-app
```

## Render a config file from environment values

Create `app.conf.tmpl`:

```text
env={{ .APP_ENV }}
database={{ .DATABASE_URL }}
```

Render it and run your app:

```bash
we -e dev.yml --template app.conf.tmpl ./my-app --config app.conf
```

See [Reference](reference.md) for all flags and file formats.
