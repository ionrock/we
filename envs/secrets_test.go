package envs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeExecutable(t *testing.T, dir, name, content string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".bat"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write fake executable: %v", err)
	}
}

func TestSecretOPProviderAndRedaction(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "op", "#!/bin/sh\necho op-secret\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	path := filepath.Join(dir, "env.yml")
	if err := os.WriteFile(path, []byte("PASSWORD: secret:op://Private/app/password\n"), 0644); err != nil {
		t.Fatal(err)
	}

	env, err := WithEnvTyped([]string{"--env", path}, dir)
	if err != nil {
		t.Fatalf("WithEnvTyped failed: %v", err)
	}
	if env["PASSWORD"].Value != "op-secret" || !env["PASSWORD"].Secret {
		t.Fatalf("expected resolved secret, got %#v", env["PASSWORD"])
	}
	if got := env.Lines(false)[0]; got != "PASSWORD=<redacted>" {
		t.Fatalf("expected redacted line, got %q", got)
	}
}

func TestOptionalSecretOmittedOnFailure(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "op", "#!/bin/sh\nexit 1\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	path := filepath.Join(dir, "env.yml")
	if err := os.WriteFile(path, []byte("TOKEN: secret?:op://Private/app/missing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	env, err := WithEnvTyped([]string{"--env", path}, dir)
	if err != nil {
		t.Fatalf("optional secret should not fail: %v", err)
	}
	if _, ok := env["TOKEN"]; ok {
		t.Fatalf("optional failed secret should be omitted: %#v", env)
	}
}

func TestSecretExpansionPropagates(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "op", "#!/bin/sh\necho pass\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	path := filepath.Join(dir, "env.yml")
	content := "- PASSWORD: secret:op://Private/app/password\n- DATABASE_URL: postgres://user:$PASSWORD@localhost/db\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	env, err := WithEnvTyped([]string{"--env", path}, dir)
	if err != nil {
		t.Fatalf("WithEnvTyped failed: %v", err)
	}
	if env["DATABASE_URL"].Value != "postgres://user:pass@localhost/db" || !env["DATABASE_URL"].Secret {
		t.Fatalf("expected secret-derived database url, got %#v", env["DATABASE_URL"])
	}
}

func TestAWSSecretsManagerJSONPath(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "aws", "#!/bin/sh\necho '{\"database\":{\"password\":\"aws-secret\"}}'\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	path := filepath.Join(dir, "env.yml")
	if err := os.WriteFile(path, []byte("PASSWORD: secret:aws-secretsmanager://prod/app?profile=dev&region=us-west-2#database.password\n"), 0644); err != nil {
		t.Fatal(err)
	}
	env, err := WithEnvTyped([]string{"--env", path}, dir)
	if err != nil {
		t.Fatalf("WithEnvTyped failed: %v", err)
	}
	if env["PASSWORD"].Value != "aws-secret" || !env["PASSWORD"].Secret {
		t.Fatalf("expected aws secret, got %#v", env["PASSWORD"])
	}
}

func TestBitwardenFields(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "bw", "#!/bin/sh\necho '{\"login\":{\"username\":\"u\",\"password\":\"p\"},\"notes\":\"n\",\"fields\":[{\"name\":\"api\",\"value\":\"key\"}]}'\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	path := filepath.Join(dir, "env.yml")
	if err := os.WriteFile(path, []byte("API_KEY: secret:bitwarden://my-item#custom.api\n"), 0644); err != nil {
		t.Fatal(err)
	}
	env, err := WithEnvTyped([]string{"--env", path}, dir)
	if err != nil {
		t.Fatalf("WithEnvTyped failed: %v", err)
	}
	if env["API_KEY"].Value != "key" || !env["API_KEY"].Secret {
		t.Fatalf("expected bitwarden secret, got %#v", env["API_KEY"])
	}
}
