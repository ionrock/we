# Secrets design

This document captures the planned design for first-class secret support in `we`.

## Goals

- Preserve metadata that a loaded value is secret.
- Resolve secrets during environment loading so failures happen before the child command starts.
- Keep the existing source model compatible with files, directories, scripts, aliases, templates, and `--envvar`.
- Allow scripts and existing tools to emit secret references with minimal changes.
- Make secret providers pluggable behind a small interface.
- Redact secret values in `we`-controlled output and logs.

## Non-goals for the first implementation

- Prevent a child process from printing secrets it receives in its environment.
- Redact arbitrary stdout/stderr from child commands.
- Implement Kubernetes integration in v1.
- Implement external executable provider plugins in v1.
- Treat provider-looking URLs as secrets without an explicit marker.

## Internal representation

The current environment representation is effectively `map[string]string`. Secret support should first introduce a typed value representation, for example:

```go
type Value struct {
    Value    string // resolved value used for env injection and templates
    Raw      string // original configured value, e.g. secret:op://vault/item/field
    Secret   bool
    Resolved bool
    Source   string // file path, script, envvar, provider URI, etc.
    Provider string // literal, op, aws-secretsmanager, bitwarden, etc.
}

type Env map[string]Value
```

The final conversion to `KEY=value` strings should happen at the command execution boundary. This preserves metadata for redaction, inspection, templates, and future integrations.

## Secret reference syntax

Secret references are explicit scalar string values:

```yaml
PASSWORD: secret:op://Private/my-app/password
OPTIONAL_TOKEN: secret?:op://Private/my-app/optional-token
```

Rules:

- `secret:<provider-uri>` is required. Resolution failure aborts loading.
- `secret?:<provider-uri>` is optional. Resolution failure omits the variable.
- Provider-looking URLs without `secret:` are literal strings.
- This syntax works in files, directories, scripts, aliases, and `--envvar` values because it only relies on scalar value parsing.

## Expansion and propagation

Values should be expanded before secret resolution:

```yaml
OP_VAULT: Private
PASSWORD: secret:op://$OP_VAULT/my-app/password
```

If a later value expands a secret value, secrecy propagates:

```yaml
PASSWORD: secret:op://Private/my-app/password
DATABASE_URL: postgres://user:$PASSWORD@localhost/db
```

`DATABASE_URL` should be marked secret because it contains secret material.

Implementation implication: replace direct `os.ExpandEnv` use in loading paths with a helper that returns both the expanded string and whether any referenced values were secret.

## Redaction and display

`we`-controlled logs and inspection output should redact secret values.

When no command is provided, `we` should stop delegating to external `env` and instead print the computed environment itself so secrets can be redacted:

```text
APP_ENV=development
DATABASE_PASSWORD=<redacted>
DATABASE_URL=<redacted>
```

Add an explicit escape hatch:

```bash
we --show-secrets
```

In `--agent` mode, `--show-secrets` must be disabled. The command should either fail fast or ignore `--show-secrets` with a warning; failing fast is preferred.

`--agent` v1 boundary: child commands still receive resolved secrets and can print them. We will not attempt stdout/stderr redaction in v1.

`--clean` semantics should remain unchanged. Without `--clean`, the child inherits the parent environment plus loaded values. With `--clean`, it receives only values loaded by `we`.

## Templates

Templates should render real secret values by default because templates are commonly used to create runtime config files required by the child command.

`we` debug output should not log rendered secret values. Rendered template targets may contain secret material and should be treated as sensitive in documentation and future `--agent` sandbox improvements.

## Provider interface

Providers should be built-in Go implementations for v1, behind a small interface so external plugins can be added later.

Example shape:

```go
type Provider interface {
    Scheme() string
    Resolve(ctx context.Context, ref Ref) (string, error)
}

type Ref struct {
    Original string
    Scheme   string
    URL      *url.URL
    Optional bool
}
```

Providers should shell out to existing CLIs in v1 where that matches user workflows and avoids reimplementing auth/session behavior.

## Provider roadmap

### Phase 1: 1Password

Use the `op` CLI.

Primary syntax:

```yaml
DATABASE_PASSWORD: secret:op://Private/my-app/password
```

Implementation should prefer:

```bash
op read op://Private/my-app/password
```

This reuses 1Password's own secret reference syntax and existing `op` authentication/session behavior.

### Phase 2: AWS Secrets Manager

Use the `aws` CLI and support JSON field extraction.

```yaml
# Whole SecretString
DATABASE_CONFIG: secret:aws-secretsmanager://prod/app

# Top-level JSON key
DATABASE_PASSWORD: secret:aws-secretsmanager://prod/app#password

# Nested JSON path
DATABASE_PASSWORD: secret:aws-secretsmanager://prod/app#database.password

# Optional profile/region
DATABASE_PASSWORD: secret:aws-secretsmanager://prod/app?profile=dev&region=us-west-2#database.password
```

Provider behavior should match the existing workflow:

```bash
aws secretsmanager get-secret-value \
  --secret-id prod/app \
  --query SecretString \
  --output text
```

Then:

- no fragment returns the whole `SecretString`
- a fragment parses `SecretString` as JSON and selects the dot path
- missing paths fail for `secret:` and omit for `secret?:`

This emulates workflows like:

```bash
aws secretsmanager get-secret-value ... | jq -r '.SecretString | fromjson | .path.to.value'
```

### Phase 3: Bitwarden

Use the `bw` CLI.

```yaml
BW_PASSWORD: secret:bitwarden://item-name#password
BW_USERNAME: secret:bitwarden://item-name#username
BW_NOTES: secret:bitwarden://item-name#notes
BW_CUSTOM: secret:bitwarden://item-name#custom.my-field
```

Provider behavior:

```bash
bw get item "item-name"
```

Then parse JSON:

- `#username` -> `.login.username`
- `#password` -> `.login.password`
- `#notes` -> `.notes`
- `#custom.foo` -> custom field named `foo`

### Later: Kubernetes

A good later integration is loading existing Kubernetes Secret values locally:

```yaml
PASSWORD: secret:kubernetes://namespace/secret-name#key
```

Provider behavior would shell out to `kubectl`:

```bash
kubectl get secret secret-name -n namespace -o json
```

Then read `.data[key]` and base64-decode it. This should be deferred until the core metadata model and initial providers are stable.

Other Kubernetes opportunities, also deferred:

- generate Kubernetes `Secret` manifests from a computed `we` environment
- generate `envFrom` / `secretRef` deployment snippets
- improve `--agent` sandbox behavior around generated secret-bearing files

## Testing strategy

### Representation and compatibility tests

- Existing YAML/JSON flattening behavior remains unchanged for ordinary scalar and nested values.
- Existing `file`, `directory`, `script`, `envvar`, `alias`, and `template` tests continue to pass.
- Values without `secret:` remain non-secret, including literal strings that look like provider URLs.
- `secret:` and `secret?:` markers are recognized from all scalar-producing sources.

### Expansion tests

- Secret URIs expand environment variables before provider resolution.
- Values derived from secret values are marked secret.
- Values derived only from non-secret values remain non-secret.
- Expansion order and override precedence match current behavior.

### Optional secret tests

- Required secret resolution failure aborts loading.
- Optional secret resolution failure omits the variable.
- Optional omission does not override an existing value in the computed environment.

### Redaction tests

- Debug logs redact values marked secret.
- No-command output redacts secrets by default.
- `--show-secrets` prints real values outside `--agent` mode.
- `--agent --show-secrets` fails fast.
- Child command output is not redacted in v1; document this boundary with a test or explicit fixture if practical.

### Provider tests

Provider tests should avoid real secret manager dependencies by using fake CLIs placed earlier in `PATH`.

1Password fake `op`:

- Assert `op read op://...` is invoked with the expected reference.
- Return a known value and verify it becomes a secret `Value`.
- Return non-zero and verify required vs optional behavior.

AWS Secrets Manager fake `aws`:

- Assert `aws secretsmanager get-secret-value --secret-id ... --query SecretString --output text` is invoked.
- Verify `profile` and `region` query params become CLI flags.
- Return a plain string and verify whole-secret extraction.
- Return JSON and verify `#topLevel` and `#nested.path` extraction.
- Verify missing JSON path behavior for required and optional refs.
- Verify malformed JSON with a fragment fails clearly.

Bitwarden fake `bw`:

- Assert `bw get item <item>` is invoked.
- Return item JSON and verify `#username`, `#password`, `#notes`, and `#custom.name` extraction.
- Verify missing fields for required and optional refs.

### Integration-style tests

- Script source can emit `PASSWORD: secret:op://...` and the resulting value is resolved and marked secret.
- `--envvar PASSWORD=secret:op://...` resolves and marks secret.
- Template rendering receives real values while `we` logs/inspection redact them.
- `--clean` preserves existing behavior with typed values converted at the boundary.
