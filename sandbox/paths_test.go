package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectWithenvArgPathsIncludesExplicitEnvFiles(t *testing.T) {
	root := t.TempDir()
	envPath := filepath.Join(root, "secret.yml")
	if err := os.WriteFile(envPath, []byte("SECRET: value\n"), 0644); err != nil {
		t.Fatalf("write env failed: %v", err)
	}

	paths := CollectWithenvArgPaths([]string{"--env", "secret.yml"}, root)
	if !containsPath(paths, envPath) {
		t.Fatalf("expected paths to contain %q, got %#v", envPath, paths)
	}
}

func TestCollectWithenvArgPathsIncludesAliasEntries(t *testing.T) {
	root := t.TempDir()
	aliasPath := filepath.Join(root, ".withenv.yml")
	envPath := filepath.Join(root, "devenv.json")
	if err := os.WriteFile(envPath, []byte(`{"SECRET":"value"}`), 0644); err != nil {
		t.Fatalf("write env failed: %v", err)
	}
	if err := os.WriteFile(aliasPath, []byte("---\n- file: devenv.json\n"), 0644); err != nil {
		t.Fatalf("write alias failed: %v", err)
	}

	paths := CollectWithenvArgPaths([]string{"--alias", ".withenv.yml"}, root)
	if !containsPath(paths, aliasPath) {
		t.Fatalf("expected paths to contain alias %q, got %#v", aliasPath, paths)
	}
	if !containsPath(paths, envPath) {
		t.Fatalf("expected paths to contain env %q, got %#v", envPath, paths)
	}
}

func TestCollectEnvrcPathsIncludesDotenvAndSourceEnv(t *testing.T) {
	root := t.TempDir()
	envrcPath := filepath.Join(root, ".envrc")
	dotenvPath := filepath.Join(root, ".env")
	localPath := filepath.Join(root, "local.env")

	if err := os.WriteFile(envrcPath, []byte("dotenv\nsource_env local.env\ndotenv_if_exists missing.env\n"), 0644); err != nil {
		t.Fatalf("write envrc failed: %v", err)
	}
	if err := os.WriteFile(dotenvPath, []byte("SECRET=value\n"), 0644); err != nil {
		t.Fatalf("write dotenv failed: %v", err)
	}
	if err := os.WriteFile(localPath, []byte("LOCAL=value\n"), 0644); err != nil {
		t.Fatalf("write local failed: %v", err)
	}

	paths, err := CollectEnvrcPaths(root)
	if err != nil {
		t.Fatalf("CollectEnvrcPaths failed: %v", err)
	}
	for _, expected := range []string{envrcPath, dotenvPath, localPath} {
		if !containsPath(paths, expected) {
			t.Fatalf("expected paths to contain %q, got %#v", expected, paths)
		}
	}
}

func containsPath(paths []string, path string) bool {
	resolved, err := ResolvePath(path)
	if err != nil {
		return false
	}
	for _, p := range paths {
		if p == resolved {
			return true
		}
	}
	return false
}
