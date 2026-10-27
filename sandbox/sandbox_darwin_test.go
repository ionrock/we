//go:build darwin

package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDarwinSandboxDeniesReadAndWrite(t *testing.T) {
	if _, err := os.Stat("/usr/bin/sandbox-exec"); err != nil {
		t.Skip("sandbox-exec not available")
	}

	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	secretDir := filepath.Join(dir, "secret")
	if err := os.Mkdir(secretDir, 0700); err != nil {
		t.Fatal(err)
	}
	secretFile := filepath.Join(secretDir, "creds")
	if err := os.WriteFile(secretFile, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	sb, _ := newPlatformSandbox(Config{Deny: []DenyRule{{Path: secretDir, Dir: true}}})

	run := func(script string) error {
		cmd, args, err := sb.Wrap("/bin/sh", []string{"-c", script})
		if err != nil {
			t.Fatal(err)
		}
		return exec.Command(cmd, args...).Run()
	}

	if err := run("true"); err != nil {
		t.Fatalf("sandbox profile failed to load: %v", err)
	}
	if err := run("cat " + secretFile); err == nil {
		t.Error("read of denied path succeeded")
	}
	if err := run("echo pwned > " + secretFile); err == nil {
		t.Error("write to denied path succeeded")
	}
	if err := run("echo pwned > " + filepath.Join(secretDir, "new")); err == nil {
		t.Error("file creation in denied dir succeeded")
	}
	if err := run("echo ok > " + filepath.Join(dir, "allowed")); err != nil {
		t.Errorf("write outside denied path failed: %v", err)
	}
}
