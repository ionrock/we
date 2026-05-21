# Secrets

`we` can resolve secrets while it loads environment values. Secret values are passed to the child command as normal environment variables, but `we` tracks their metadata so inspection output and debug logs can redact them.

## Secret reference syntax

Use a scalar string value prefixed with `secret:`:

```yaml
DATABASE_PASSWORD: "secret:op://Private/my-app/password"
```

Use `secret?:` for optional secrets. If resolution fails, the variable is omitted instead of aborting the command:

```yaml
OPTIONAL_TOKEN: "secret?:op://Private/my-app/optional-token"
```

Rules:

- `secret:` is required for provider resolution. A value that merely looks like `op://...` is treated as a literal string.
- Required secret failures abort environment loading before the child command starts.
- Optional secret failures omit that variable.
- Secret refs work anywhere `we` loads scalar environment values through withenv sources: YAML/JSON files, environment directories, script output, and `--envvar` values.
- Quote values in YAML when they contain spaces or characters that YAML might otherwise interpret.

## 1Password examples

The `op` provider uses the 1Password CLI and reads the reference with `op read`.

```yaml
OPENAI_API_KEY: "secret:op://Private/api.openai.default.OPENAI_API_KEY/api key"
GEMINI_PROJECT_ID: "secret:op://Private/api.gemini.project.GEMINI_PROJECT_ID/project id"
```

This resolves by running commands equivalent to:

```bash
op read 'op://Private/api.openai.default.OPENAI_API_KEY/api key'
op read 'op://Private/api.gemini.project.GEMINI_PROJECT_ID/project id'
```

You can keep machine-wide secrets in `~/.withenv_global.yml`:

```yaml title="~/.withenv_global.yml"
---
ANTHROPIC_API_KEY: "secret:op://Private/api.anthropic.default.ANTHROPIC_API_KEY/api key"
OPENAI_API_KEY: "secret:op://Private/api.openai.default.OPENAI_API_KEY/api key"
LINEAR_API_KEY: "secret:op://Private/api.linear.default.LINEAR_API_KEY/api key"
```

Or project-specific secrets in a file referenced by `.withenv.yml`:

```yaml title=".withenv.yml"
- file: secrets.yml
```

```yaml title="secrets.yml"
DATABASE_PASSWORD: "secret:op://Private/my-app/database-password"
```

## AWS Secrets Manager examples

The `aws-secretsmanager` provider shells out to the AWS CLI:

```bash
aws secretsmanager get-secret-value \
  --secret-id prod/my-app \
  --query SecretString \
  --output text
```

Use the whole `SecretString`:

```yaml
DATABASE_CONFIG: "secret:aws-secretsmanager://prod/my-app"
```

Select a field from a JSON `SecretString` with a fragment:

```yaml
DATABASE_PASSWORD: "secret:aws-secretsmanager://prod/my-app#password"
DATABASE_PASSWORD: "secret:aws-secretsmanager://prod/my-app#database.password"
```

Pass profile or region as query parameters:

```yaml
DATABASE_PASSWORD: "secret:aws-secretsmanager://prod/my-app?profile=dev&region=us-west-2#database.password"
```

## Bitwarden examples

The `bitwarden` provider shells out to `bw get item <item>` and extracts common fields:

```yaml
BW_USERNAME: "secret:bitwarden://my-login#username"
BW_PASSWORD: "secret:bitwarden://my-login#password"
BW_NOTES: "secret:bitwarden://my-login#notes"
BW_API_KEY: "secret:bitwarden://my-login#custom.api-key"
```

## Expansion and propagation

Values are expanded before secret resolution, so refs can reuse earlier environment values:

```yaml
OP_VAULT: Private
DATABASE_PASSWORD: "secret:op://$OP_VAULT/my-app/database-password"
```

If a later value expands a secret value, the derived value is treated as secret too:

```yaml
DATABASE_PASSWORD: "secret:op://Private/my-app/database-password"
DATABASE_URL: "postgres://app:$DATABASE_PASSWORD@localhost/app"
```

`DATABASE_URL` is redacted in `we` output because it contains `DATABASE_PASSWORD`.

When ordering matters within one file, use a list of maps:

```yaml
---
- OP_VAULT: Private
- DATABASE_PASSWORD: "secret:op://$OP_VAULT/my-app/database-password"
- DATABASE_URL: "postgres://app:$DATABASE_PASSWORD@localhost/app"
```

## Inspecting secret-loaded environments

With no command, `we` prints the computed environment. Secret values are redacted by default:

```bash
we --clean
```

```text
DATABASE_PASSWORD=<redacted>
DATABASE_URL=<redacted>
```

Use `--show-secrets` to inspect actual values:

```bash
we --show-secrets --clean
```

`--show-secrets` cannot be used with `--agent`.

## Using secrets from `--envvar` and scripts

One-off secret refs work with `--envvar`:

```bash
we --envvar 'OPENAI_API_KEY=secret:op://Private/api.openai.default.OPENAI_API_KEY/api key' your-command
```

Scripts can emit YAML or JSON containing secret refs. `we` resolves them after the script output is parsed:

```bash
we --script './print-secret-refs' your-command
```

Example script output:

```yaml
API_TOKEN: "secret:op://Private/my-app/api-token"
```

## Limitations

- `.envrc`, `.env`, and `source_env` files are parsed for literal assignments only. Secret provider refs in these direnv-style files are not currently resolved; they will be passed through as literal strings. Use a withenv YAML/JSON file, environment directory, script source, `--envvar`, or `~/.withenv_global.yml` for secret refs.
- A top-level `secrets:` section has no special meaning. Nested YAML is flattened into env var names, so put secret refs directly under the env var names you want to set.
- `we` redacts its own inspection and debug output, but it cannot prevent the child process from printing secrets it receives in the environment.
- Provider CLIs must be installed and authenticated before `we` runs (`op`, `aws`, or `bw`). `we` reuses those tools' existing auth/session behavior.
- Provider-looking values without `secret:` are literals by design.
