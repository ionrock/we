package envs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFindEnvrcWalksUp(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	sub := filepath.Join(project, "a", "b")

	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, ".envrc"), []byte("export FOO=bar\n"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got, err := findEnvrc(sub)
	if err != nil {
		t.Fatalf("findEnvrc failed: %v", err)
	}

	expected := filepath.Join(project, ".envrc")
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestFindEnvrcNotFound(t *testing.T) {
	root := t.TempDir()
	got, err := findEnvrc(root)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got path=%q err=%v", got, err)
	}
}

func TestEnvrcParse(t *testing.T) {
	root := t.TempDir()
	envrcPath := filepath.Join(root, ".envrc")
	content := `# comment
export FOO=bar
BAR="hello world"
export EMPTY=
source_env_if_exists .env.local
`
	if err := os.WriteFile(envrcPath, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	env, err := Envrc{path: envrcPath}.Parse()
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if env["FOO"] != "bar" {
		t.Fatalf("expected FOO=bar, got %q", env["FOO"])
	}
	if env["BAR"] != "hello world" {
		t.Fatalf("expected BAR=hello world, got %q", env["BAR"])
	}
	if env["EMPTY"] != "" {
		t.Fatalf("expected EMPTY='', got %q", env["EMPTY"])
	}
	if _, ok := env["source_env_if_exists"]; ok {
		t.Fatalf("unexpected parsed key from non-assignment line")
	}
}

func TestMaybeLoadEnvrcAppliesVars(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	sub := filepath.Join(project, "sub")

	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, ".envrc"), []byte("export FOO=bar\n"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	t.Setenv("WE_NO_DIRENV", "")
	changes, err := MaybeLoadEnvrc(sub, false)
	if err != nil {
		t.Fatalf("MaybeLoadEnvrc failed: %v", err)
	}
	if changes["FOO"] != "bar" {
		t.Fatalf("expected FOO=bar, got %q", changes["FOO"])
	}
	if os.Getenv("FOO") != "bar" {
		t.Fatalf("expected process env FOO=bar, got %q", os.Getenv("FOO"))
	}
}

func TestMaybeLoadEnvrcDisabled(t *testing.T) {
	root := t.TempDir()
	if _, err := MaybeLoadEnvrc(root, true); err != nil {
		t.Fatalf("expected no error when disabled by flag, got: %v", err)
	}

	t.Setenv("WE_NO_DIRENV", "1")
	if _, err := MaybeLoadEnvrc(root, false); err != nil {
		t.Fatalf("expected no error when disabled by env var, got: %v", err)
	}
}
