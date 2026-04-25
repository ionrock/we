package sandbox

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

// CollectWithenvPaths parses a .withenv.yml alias file and recursively
// finds all file paths it references. Returns absolute, symlink-resolved paths.
// The alias file itself is included in the result.
func CollectWithenvPaths(aliasPath string) ([]string, error) {
	resolved, err := ResolvePath(aliasPath)
	if err != nil {
		return nil, err
	}

	paths := []string{resolved}

	b, err := os.ReadFile(aliasPath)
	if err != nil {
		return paths, err
	}

	var entries []map[string]string
	if err := yaml.Unmarshal(b, &entries); err != nil {
		// Try flexible parsing for files with sandbox: entries
		var flexEntries []map[string]interface{}
		if err2 := yaml.Unmarshal(b, &flexEntries); err2 != nil {
			return paths, err
		}
		// Extract only string-valued entries
		for _, entry := range flexEntries {
			for k, v := range entry {
				if sv, ok := v.(string); ok {
					entries = append(entries, map[string]string{k: sv})
				}
			}
		}
	}

	dir := filepath.Dir(aliasPath)

	for _, entry := range entries {
		for k, v := range entry {
			var entryPath string

			switch k {
			case "file", "env":
				entryPath = fileLocalPath(dir, v)
			case "directory", "dir":
				entryPath = fileLocalPath(dir, v)
			case "script":
				// If the script value looks like a file path, add it
				if isFilePath(v) {
					entryPath = fileLocalPath(dir, v)
				}
			case "alias":
				// Recurse into nested alias files
				nestedPath := fileLocalPath(dir, v)
				nested, err := CollectWithenvPaths(nestedPath)
				if err != nil {
					log.Debug().Err(err).Msgf("skipping nested alias %s", nestedPath)
					continue
				}
				paths = append(paths, nested...)
				continue
			default:
				continue
			}

			if entryPath == "" {
				continue
			}

			resolvedEntry, err := ResolvePath(entryPath)
			if err != nil {
				log.Debug().Err(err).Msgf("skipping path %s", entryPath)
				continue
			}
			paths = append(paths, resolvedEntry)
		}
	}

	return paths, nil
}

// CollectEnvrcPath finds the .envrc file starting from startDir and walking up.
// Returns the resolved path, or empty string if not found.
func CollectEnvrcPath(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	root, err := filepath.Abs("/")
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, ".envrc")
		if _, err := os.Stat(candidate); err == nil {
			return ResolvePath(candidate)
		}

		if dir == root {
			break
		}
		dir = filepath.Dir(dir)
	}

	return "", nil
}

// fileLocalPath resolves a path relative to a base directory.
func fileLocalPath(baseDir string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}

// isFilePath heuristically determines if a string looks like a file path
// rather than a complex command.
func isFilePath(s string) bool {
	// Contains spaces or pipes? Probably a command, not a file path.
	for _, c := range s {
		if c == ' ' || c == '|' || c == ';' || c == '&' {
			return false
		}
	}
	return true
}
