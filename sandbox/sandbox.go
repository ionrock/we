package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
)

// DenyRule represents a path to deny access to.
type DenyRule struct {
	Path string // Absolute, symlink-resolved path
	Dir  bool   // true = subpath (directory), false = literal (file)
}

// AllowRule represents an exception to a deny rule.
type AllowRule struct {
	Path string // Absolute, symlink-resolved path
	Dir  bool
}

// Config holds the complete sandbox configuration.
type Config struct {
	Deny    []DenyRule
	Allow   []AllowRule
	DenyNet bool
}

// Sandbox wraps a child command with OS-level filesystem restrictions.
type Sandbox interface {
	// Wrap takes the original command and args, returns a new command
	// and args that will execute under the sandbox.
	Wrap(cmd string, args []string) (string, []string, error)
}

// DefaultDenyPaths returns the standard set of sensitive paths to deny.
// Paths that don't exist on the current system are silently skipped.
func DefaultDenyPaths() []DenyRule {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	candidates := []struct {
		rel string
		dir bool
	}{
		{".ssh", true},
		{".aws", true},
		{".gnupg", true},
		{".config/gcloud", true},
		{".azure", true},
		{".config/op", true},
		{".netrc", false},
		{".npmrc", false},
		{".pypirc", false},
		{filepath.Join(".docker", "config.json"), false},
	}

	var rules []DenyRule
	for _, c := range candidates {
		abs := filepath.Join(home, c.rel)
		resolved, err := ResolvePath(abs)
		if err != nil {
			// Path doesn't exist, skip
			continue
		}
		rules = append(rules, DenyRule{Path: resolved, Dir: c.dir})
	}

	return rules
}

// ResolvePath resolves symlinks and returns the absolute path.
// Returns an error if the path does not exist.
func ResolvePath(path string) (string, error) {
	// Expand ~ if present
	path = ExpandTilde(path)

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolving symlinks for %s: %w", abs, err)
	}

	return resolved, nil
}

// New returns the platform-appropriate Sandbox implementation.
func New(cfg Config) (Sandbox, error) {
	return newPlatformSandbox(cfg)
}

// ExpandTilde replaces a leading ~ with the user's home directory.
func ExpandTilde(path string) string {
	if len(path) == 0 {
		return path
	}

	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if len(path) == 1 {
			return home
		}
		if path[1] == '/' {
			return filepath.Join(home, path[2:])
		}
	}

	return path
}
