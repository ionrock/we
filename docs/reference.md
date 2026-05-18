# Reference

## CLI synopsis

```text
we [global options] [COMMAND]
we convert [command options] <input-file>
```

If `COMMAND` is omitted, `we` runs `env` so you can inspect the computed environment.

## Global flags

| Flag | Repeatable | Description |
| --- | --- | --- |
| `--env FILE`, `-e FILE` | yes | Load variables from a YAML or JSON file. |
| `--script COMMAND`, `-s COMMAND` | yes | Run a command and load its stdout as YAML or JSON. |
| `--directory DIR`, `-d DIR` | yes | Recursively load `.yaml`, `.yml`, and `.json` files from a directory. |
| `--alias FILE`, `-a FILE` | yes | Load an alias YAML file containing `file`, `directory`, `script`, `envvar`, `template`, or `sandbox` entries. |
| `--envvar KEY=VALUE`, `-E KEY=VALUE` | yes | Set or override one variable. |
| `--template TEMPLATE[:TARGET]`, `-t TEMPLATE[:TARGET]` | yes | Render a Go template before running the command. Without `:TARGET`, the target is the template path with `.tmpl` removed. |
| `--clean`, `-c` | no | Run the command with only variables loaded by withenv sources. |
| `--no-direnv` | no | Disable automatic upward search and loading of `.envrc`. |
| `--debug`, `-D` | no | Enable debug logging. |
| `--agent` | no | Run with agent protections: filesystem sandbox plus Claude Code command re-evaluation hooks. |
| `--sandbox-deny PATH` | yes | Deny additional paths in the agent sandbox. |
| `--sandbox-allow PATH` | yes | Allow paths as exceptions to sandbox deny rules. |
| `--sandbox-deny-network` | no | Also restrict network access in the agent sandbox. |
| `--help`, `-h` | no | Show help. |
| `--version`, `-v` | no | Print the version. |

## `convert` command flags

| Flag | Description |
| --- | --- |
| `--output FILE`, `-o FILE` | Write converted YAML to `FILE` instead of stdout. |
| `--debug`, `-D` | Enable debug logging. |
| `--help`, `-h` | Show help. |

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

String values are expanded with the current process environment. A value that starts and ends with a backtick is executed as a command, and stdout becomes the value:

```yaml
---
- TOKEN: "`vault read -field=token secret/app`"
```

Command values are split lexically and support simple pipelines with `|`; they are not evaluated by a shell.

## Alias file format

An alias file is a YAML list. Supported entries are:

```yaml
---
- file: env/base.yml        # same as --env
- directory: env/dev        # same as --directory
- script: ./env-from-vault  # same as --script
- envvar: APP_ENV=dev       # same as --envvar
- template: app.conf.tmpl   # same as --template
- sandbox:
    deny:
      - ~/.kube
    allow:
      - ~/.kube/cache
    deny-network: true
    skip-prefix:
      - git
```

Relative `file`, `directory`, and `template` paths are resolved relative to the alias file. `script` and `envvar` values are not path-resolved.

## Automatic files and precedence

`we` always searches upward from the current directory for `.withenv.yml`. If found, it is loaded as an alias before explicit CLI environment flags.

`we` also loads `~/.withenv_global.yml` when present and searches upward for `.envrc` unless `--no-direnv` is set or `WE_NO_DIRENV=1` is present. `.envrc` parsing supports assignment lines such as `KEY=value` and `export KEY=value`; shell code is not executed.

Withenv sources are applied in this order, and later sources override earlier ones:

1. `~/.withenv_global.yml`
2. nearest `.envrc`
3. nearest `.withenv.yml` alias
4. explicit CLI flags from left to right

Without `--clean`, the child command also inherits the rest of your shell environment. With `--clean`, only variables set by withenv sources are passed to the command.

## Templates

Templates use Go's `text/template` package plus Sprig functions. The template data is the environment map, so both of these forms are valid:

```text
port={{ .PORT }}
port={{ env "PORT" }}
```

## Agent sandbox

`--agent` wraps the command in the platform sandbox implementation and writes temporary Claude Code hooks under `.claude/settings.local.json` so Bash tool commands are prefixed with `we` and re-evaluate the environment.

Default denied locations include common sensitive files and directories such as `~/.ssh`, `~/.aws`, `~/.gnupg`, cloud CLI config, `~/.netrc`, package publishing credentials, and Docker config. Additional deny/allow rules can be provided through CLI flags or a `sandbox` entry in `.withenv.yml`.
