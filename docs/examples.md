# Examples

## Application configuration

Keep environment-specific config in the repository:

```text
my-app/
  env/
    base.yml
    dev.yml
    prod.yml
  .withenv.yml
```

```yaml title="env/base.yml"
---
APP_NAME: my-app
PORT: "8080"
LOG_LEVEL: info
```

```yaml title="env/dev.yml"
---
LOG_LEVEL: debug
DATABASE_URL: postgres://localhost/my_app
```

```yaml title=".withenv.yml"
---
- file: env/base.yml
- file: env/dev.yml
```

Run locally:

```bash
we ./my-app
```

Run with production values explicitly:

```bash
we -e env/base.yml -e env/prod.yml ./my-app
```

## Makefile workflow

```make
ENV ?= dev

run:
	we -a env/$(ENV).alias.yml ./my-app

print-env:
	we --clean -a env/$(ENV).alias.yml
```

Then:

```bash
make run
make ENV=prod print-env
```

## Docker entrypoint

Package config with the image and use the same command locally and in the container:

```dockerfile
FROM ubuntu:24.04
COPY we /usr/local/bin/we
COPY my-app /usr/local/bin/my-app
COPY env /opt/my-app/env
WORKDIR /opt/my-app
CMD ["we", "-a", "env/prod.alias.yml", "my-app"]
```

## Dynamic secrets

Load a token and then use it in another script source:

```yaml
---
- ROLE_ID: my-role
- VAULT_TOKEN: "`vault write -format=json auth/approle/login role_id=$ROLE_ID secret_id=$SECRET_ID | jq -r .auth.client_token`"
```

```bash
we -e vault.yml \
   --script 'vault kv get -format=json secret/my-app | jq .data.data' \
   ./my-app
```

## Template rendering

```yaml title="env.yml"
---
LISTEN: 10.0.0.1:8900
CLUSTER_HOSTS:
  - 10.0.0.2
  - 10.0.0.3
  - 10.0.0.4
```

```text title="service.ini.tmpl"
[service]
listen = {{ .LISTEN }}
hosts = {{ .CLUSTER_HOSTS }}
```

```bash
we -e env.yml -t service.ini.tmpl cat service.ini
```

## AI agent sandbox

Run an agent with configuration loaded, default sensitive files denied, and Claude Code Bash tool commands automatically re-prefixed with `we`:

```bash
we --agent -e env/dev.yml claude
```

Add custom sandbox policy in `.withenv.yml`:

```yaml
---
- file: env/dev.yml
- sandbox:
    deny:
      - ~/.kube
    allow:
      - ~/.kube/cache
    deny-network: true
    skip-prefix:
      - git
```
