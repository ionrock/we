package envs

import (
	"os"
	"path/filepath"
	"testing"

	age "filippo.io/age"
	"github.com/ionrock/we/sopsutil"
)

func TestSOPSYAMLDecryptsAndRedacts(t *testing.T) {
	dir := t.TempDir()
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate age identity: %v", err)
	}
	t.Setenv("SOPS_AGE_KEY", identity.String())

	plain := []byte("DATABASE_PASSWORD: super-secret\nLOG_LEVEL: debug\n")
	encrypted, err := sopsutil.EncryptYAML(filepath.Join(dir, "secrets.enc.yml"), plain, sopsutil.EncryptOptions{
		AgeRecipients: []string{identity.Recipient().String()},
	})
	if err != nil {
		t.Fatalf("encrypt fixture: %v", err)
	}
	path := filepath.Join(dir, "secrets.enc.yml")
	if err := os.WriteFile(path, encrypted, 0600); err != nil {
		t.Fatal(err)
	}

	env, err := WithEnvTyped([]string{"--env", path}, dir)
	if err != nil {
		t.Fatalf("WithEnvTyped failed: %v", err)
	}
	if got := env["DATABASE_PASSWORD"].Value; got != "super-secret" {
		t.Fatalf("expected decrypted password, got %q", got)
	}
	if got := env["LOG_LEVEL"].Value; got != "debug" {
		t.Fatalf("expected decrypted log level, got %q", got)
	}
	if !env["LOG_LEVEL"].Secret {
		t.Fatalf("all values from SOPS files should be marked secret")
	}
	if got := env.Lines(false); got[0] != "DATABASE_PASSWORD=<redacted>" || got[1] != "LOG_LEVEL=<redacted>" {
		t.Fatalf("expected redacted SOPS values, got %#v", got)
	}
}
