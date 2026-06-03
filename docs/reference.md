# Reference

## CLI synopsis

```text
we [global options] [COMMAND]
we convert [command options] <input-file>
we sops encrypt [command options] <input-file>
we sops decrypt [command options] <input-file>
```

If `COMMAND` is omitted, `we` prints the computed environment so you can inspect it. Values known or inferred to be secret are redacted by default.

## How sources are applied

Withenv sources are applied in this order. Later sources override earlier ones:

1. `~/.withenv_global.yml`, if present
2. nearest `.envrc`, unless disabled
3. nearest `.withenv.yml` alias
4. explicit CLI flags, from left to right

Without `--clean`, the child command also inherits the rest of your shell environment. With `--clean`, only variables set by withenv sources are passed to the command.

## Global flags

### `--env FILE`, `-e FILE`

Loads environment variables from a YAML or JSON file. Repeatable.

```bash
we -e base.yml -e dev.yml ./my-app
```

Use this when you want direct control over which files are loaded for a command. Files are applied in the order provided.

### `--directory DIR`, `-d DIR`

Recursively loads `.yaml`, `.yml`, and `.json` files from a directory. Repeatable.

```bash
we -d env/dev ./my-app
```

Use this when an environment is naturally split across multiple files.

### `--alias FILE`, `-a FILE`

Loads a withenv alias YAML file. Repeatable.

```bash
we -a env/dev.alias.yml ./my-app
```

Alias files can contain entries for `file`, `directory`, `script`, `envvar`, `template`, and `sandbox`:

```yaml
---
- file: base.yml
- directory: services
- envvar: APP_ENV=development
```

Relative `file`, `directory`, and `template` paths are resolved relative to the alias file. Use aliases for repeatable project or environment presets.

### `--envvar KEY=VALUE`, `-E KEY=VALUE`

Sets or overrides a single environment variable. Repeatable.

```bash
we -E LOG_LEVEL=trace -E PORT=9000 ./my-app
```

Use this for temporary overrides, CI-provided values, or values that should not be committed.

### `--script COMMAND`, `-s COMMAND`

Runs `COMMAND` and loads its stdout as YAML or JSON. Repeatable.

```bash
we --script 'op read op://app/dev/env.json' ./my-app
```

Use this when another tool can produce environment data, such as a secrets manager, cloud CLI, or custom script. See [Advanced usage](advanced.md#scripts) for more examples.

### `--template TEMPLATE[:TARGET]`, `-t TEMPLATE[:TARGET]`

Renders a Go template before running the command. Repeatable.

```bash
we -e devenv.yml -t app.conf.tmpl ./my-app
we -e devenv.yml -t 'app.conf.tmpl:/etc/my-app/app.conf' ./my-app
```

Without `:TARGET`, the output path is the template path with `.tmpl` removed. Use this for tools that require config files instead of environment variables.

### `--clean`, `-c`

Runs the command with only variables loaded by withenv sources.

```bash
we --clean ./my-app
we --clean -e devenv.yml
```

Use this to debug the exact environment withenv creates or to avoid accidental dependencies on your shell environment.

### `--show-secrets`

Shows secret values in no-command inspection output. This is disabled with `--agent`.

### `--no-direnv`

Disables automatic upward search and loading of `.envrc`.

```bash
we --no-direnv ./my-app
```

Use this when you want a command to ignore local `.envrc` files. You can also set `WE_NO_DIRENV=1`.

### `--debug`, `-D`

Enables debug logging.

```bash
we --debug -e devenv.yml ./my-app
```

Use this to see how sources are discovered and applied.

### `--agent`

Runs the command with AI-agent protections: filesystem sandboxing plus Claude Code command re-evaluation hooks.

```bash
we --agent claude
```

Use this when launching coding agents that should receive project environment variables while being restricted from sensitive local files. See [Advanced usage](advanced.md#agent-mode).

### `--sandbox-deny PATH`

Adds a path to deny in the agent sandbox. Repeatable. Only applies with `--agent`.

```bash
we --agent --sandbox-deny ~/.kube claude
```

Use this to block additional credentials, config directories, or project-specific sensitive paths.

### `--sandbox-allow PATH`

Adds an allow exception to sandbox deny rules. Repeatable. Only applies with `--agent`.

```bash
we --agent --sandbox-deny ~/.kube --sandbox-allow ~/.kube/cache claude
```

Use this when a broad deny rule blocks a safe subpath that the agent needs.

### `--sandbox-deny-network`

Also restricts network access in the agent sandbox. Only applies with `--agent`.

```bash
we --agent --sandbox-deny-network claude
```

Use this for offline or more tightly controlled agent sessions.

### `--help`, `-h`

Shows help.

```bash
we --help
we convert --help
```

### `--version`, `-v`

Prints the version.

```bash
we --version
```

## `sops` commands

`we sops encrypt` and `we sops decrypt` encrypt and decrypt YAML using the SOPS Go library directly. Age recipients are supported with `--age`, and decryption uses standard SOPS Age key discovery (`SOPS_AGE_KEY`, `SOPS_AGE_KEY_FILE`, and `~/.config/sops/age/keys.txt`).

```bash
we sops encrypt --age age1example... --output secrets.enc.yml secrets.yml
we sops decrypt --output secrets.yml secrets.enc.yml
```

SOPS-encrypted YAML files are also decrypted automatically when loaded with `--env`, `.withenv.yml`, or `--directory`; all values from those files are redacted by default.

### `we sops encrypt` flags

- `--age RECIPIENT`: Age recipient to encrypt to. Repeatable. If omitted, `we` attempts to use a matching `.sops.yaml` creation rule.
- `--output FILE`, `-o FILE`: write encrypted YAML to `FILE` instead of stdout.
- `--in-place`: replace the input file with encrypted YAML.
- `--encrypted-regex REGEX`: encrypt only matching keys.
- `--unencrypted-regex REGEX`: leave matching keys unencrypted.
- `--debug`, `-D`: enable debug logging.

### `we sops decrypt` flags

- `--output FILE`, `-o FILE`: write plaintext YAML to `FILE` instead of stdout.
- `--debug`, `-D`: enable debug logging.

## `convert` command

Converts a dotenv-style env script (`.env`, `.envrc`, etc.) into withenv YAML.

```bash
we convert .env
we convert .env --output devenv.yml
```

### `--output FILE`, `-o FILE`

Writes converted YAML to `FILE` instead of stdout.

### `--debug`, `-D`

Enables debug logging for conversion.

## Environment file format

YAML and JSON files may contain a map or a list of maps.

```yaml
---
APP_ENV: development
PORT: "8080"
```

A list of maps preserves ordering inside one file and allows a later value to use an earlier value:

```yaml
---
- HOST: localhost
- URL: http://$HOST:8080
```

Nested objects are flattened with underscores, and lists are concatenated with spaces:

```yaml
---
database:
  host: localhost
hosts:
  - app1
  - app2
```

This produces `DATABASE_HOST=localhost` and `HOSTS="app1 app2"`.

### SOPS-encrypted YAML

If a YAML source contains top-level SOPS metadata, `we` decrypts it before flattening and loading values. This works for `--env`, alias `file` entries, and YAML files found by `--directory`.

### Secret references

YAML/JSON environment values can use `secret:` references. `we` resolves these before running the child command and redacts them in inspection output:

```yaml
PAYMENTS_API_KEY: "secret:op://Private/services.payments.dev/API key"
DATABASE_PASSWORD: "secret:aws-secretsmanager://prod/my-app#database.password"
OPTIONAL_TOKEN: "secret?:op://Private/my-app/optional-token"
```

`secret:` is required and fails if the provider lookup fails. `secret?:` is optional and omits the variable on failure. See [Secrets](secrets.md) for provider syntax, examples, and limitations.

## Automatic files

### `.withenv.yml`

`we` searches upward from the current directory for `.withenv.yml`. If found, it is loaded as an alias before explicit CLI environment flags.

### `~/.withenv_global.yml`

If present, this file is loaded before project sources. Use it for machine-wide defaults.

### `.envrc`

`we` searches upward for `.envrc` and parses assignment lines like:

```bash
export APP_ENV=development
DATABASE_URL=postgres://localhost/myapp
```

For direnv compatibility, `we` also supports these stdlib-style loading directives without executing arbitrary shell code:

```bash
dotenv                  # load .env next to the .envrc
dotenv .env.local       # load a specific dotenv file
dotenv_if_exists .env   # load only if present
source_env env.sh       # parse another assignment file
source_env_if_exists local.env
```

`.env` files are not auto-loaded by themselves. Add `dotenv` to `.envrc` when you want direnv-style `.env` loading. Disable `.envrc` behavior with `--no-direnv` or `WE_NO_DIRENV=1`.

Secret provider refs are not resolved from `.envrc`, `.env`, or `source_env` files. Use withenv YAML/JSON sources, environment directories, scripts, `--envvar`, or `~/.withenv_global.yml` for `secret:` values.
