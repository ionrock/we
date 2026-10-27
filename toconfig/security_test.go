package toconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteWritesPrivateFileByDefault(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "app.cfg.tmpl")
	target := filepath.Join(dir, "app.cfg")
	if err := os.WriteFile(tmpl, []byte("secret={{ .WE_TEST_SECRET }}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WE_TEST_SECRET", "hunter2")

	if err := ApplyTemplate(tmpl); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("new rendered file mode = %o, want 600", perm)
	}
}

func TestExecutePreservesExistingMode(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "app.cfg.tmpl")
	target := filepath.Join(dir, "app.cfg")
	os.WriteFile(tmpl, []byte("ok\n"), 0644)
	os.WriteFile(target, []byte("old\n"), 0640)
	os.Chmod(target, 0640)

	if err := ApplyTemplate(tmpl); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(target)
	if perm := info.Mode().Perm(); perm != 0640 {
		t.Errorf("mode = %o, want existing 640", perm)
	}
}

func TestExecuteFailedRenderKeepsTarget(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "bad.tmpl")
	target := filepath.Join(dir, "bad")
	os.WriteFile(tmpl, []byte("{{ .Broken "), 0644)
	os.WriteFile(target, []byte("original"), 0600)

	if err := ApplyTemplate(tmpl); err == nil {
		t.Fatal("expected template parse error")
	}
	b, _ := os.ReadFile(target)
	if string(b) != "original" {
		t.Errorf("target was modified on failure: %q", b)
	}
}

func TestExecuteRefusesToOverwriteTemplate(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "config")
	os.WriteFile(tmpl, []byte("keep me"), 0644)

	if err := ApplyTemplate(tmpl); err == nil {
		t.Fatal("expected error when template and target are the same file")
	}
	b, _ := os.ReadFile(tmpl)
	if string(b) != "keep me" {
		t.Errorf("template was overwritten: %q", b)
	}
}

func TestExecuteDoesNotFollowTargetSymlink(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim")
	os.WriteFile(victim, []byte("untouched"), 0644)
	tmpl := filepath.Join(dir, "out.tmpl")
	target := filepath.Join(dir, "out")
	os.WriteFile(tmpl, []byte("rendered"), 0644)
	if err := os.Symlink(victim, target); err != nil {
		t.Skip("symlinks unsupported")
	}

	if err := ApplyTemplate(tmpl); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(victim); string(b) != "untouched" {
		t.Errorf("symlink target was overwritten: %q", b)
	}
	if b, _ := os.ReadFile(target); string(b) != "rendered" {
		t.Errorf("target = %q, want rendered", b)
	}
}
