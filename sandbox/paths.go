package sandbox

import (
	"os"
	"path/filepath"
	"strings"

	shlex "github.com/flynn/go-shlex"
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

// CollectWithenvArgPaths finds environment-bearing file paths referenced by
// command-line withenv flags. Returned paths are absolute, symlink-resolved
// files or directories that should be hidden from agent-mode child commands.
func CollectWithenvArgPaths(args []string, baseDir string) []string {
	var paths []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		flag, value, hasInlineValue := splitFlagValue(arg)

		if flag == "" {
			continue
		}

		if isBoolFlag(flag) {
			continue
		}

		if !hasInlineValue {
			if i+1 >= len(args) {
				break
			}
			value = args[i+1]
			i++
		}

		var candidates []string
		switch flag {
		case "--env", "-e":
			candidates = append(candidates, fileLocalPath(baseDir, value))
		case "--directory", "--dir", "-d":
			candidates = append(candidates, fileLocalPath(baseDir, value))
		case "--alias", "-a":
			aliasPath := fileLocalPath(baseDir, value)
			aliasPaths, err := CollectWithenvPaths(aliasPath)
			if err != nil {
				log.Debug().Err(err).Msgf("skipping alias paths %s", aliasPath)
				candidates = append(candidates, aliasPath)
			} else {
				paths = append(paths, aliasPaths...)
			}
		case "--script", "-s":
			if isFilePath(value) {
				candidates = append(candidates, fileLocalPath(baseDir, value))
			}
		case "--sandbox-deny", "--sandbox-allow", "--envvar", "-E", "--template", "-t":
			continue
		default:
			// Unknown flag or command args: not a withenv source path.
			continue
		}

		for _, candidate := range candidates {
			resolved, err := ResolvePath(candidate)
			if err != nil {
				log.Debug().Err(err).Msgf("skipping path %s", candidate)
				continue
			}
			paths = append(paths, resolved)
		}
	}

	return paths
}

// CollectEnvrcPath finds the .envrc file starting from startDir and walking up.
// Returns the resolved path, or empty string if not found.
func CollectEnvrcPath(startDir string) (string, error) {
	paths, err := CollectEnvrcPaths(startDir)
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}

// CollectEnvrcPaths finds the .envrc file starting from startDir and walking up,
// plus any dotenv/source_env files it references. Returned paths are absolute,
// symlink-resolved paths. Missing optional files are skipped.
func CollectEnvrcPaths(startDir string) ([]string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}

	root, err := filepath.Abs("/")
	if err != nil {
		return nil, err
	}

	for {
		candidate := filepath.Join(dir, ".envrc")
		if _, err := os.Stat(candidate); err == nil {
			return collectEnvScriptPaths(candidate, map[string]bool{})
		}

		if dir == root {
			break
		}
		dir = filepath.Dir(dir)
	}

	return nil, nil
}

func collectEnvScriptPaths(path string, seen map[string]bool) ([]string, error) {
	resolved, err := ResolvePath(path)
	if err != nil {
		return nil, err
	}

	if seen[resolved] {
		return nil, nil
	}
	seen[resolved] = true

	paths := []string{resolved}
	b, err := os.ReadFile(resolved)
	if err != nil {
		return paths, err
	}

	baseDir := filepath.Dir(resolved)
	for _, line := range strings.Split(string(b), "\n") {
		parts, err := shlex.Split(line)
		if err != nil || len(parts) == 0 {
			continue
		}

		cmd := parts[0]
		if cmd != "dotenv" && cmd != "dotenv_if_exists" && cmd != "source_env" && cmd != "source_env_if_exists" {
			continue
		}

		target := ".env"
		if len(parts) > 1 {
			target = parts[1]
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(baseDir, target)
		}

		if _, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) && (cmd == "dotenv_if_exists" || cmd == "source_env_if_exists") {
				continue
			}
			log.Debug().Err(err).Msgf("skipping env script path %s", target)
			continue
		}

		nested, err := collectEnvScriptPaths(target, seen)
		if err != nil {
			log.Debug().Err(err).Msgf("skipping nested env script path %s", target)
			continue
		}
		paths = append(paths, nested...)
	}

	return paths, nil
}

func splitFlagValue(arg string) (string, string, bool) {
	if !strings.HasPrefix(arg, "-") {
		return "", "", false
	}

	parts := strings.SplitN(arg, "=", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return arg, "", false
}

func isBoolFlag(flag string) bool {
	switch flag {
	case "--debug", "-D", "--clean", "-c", "--no-direnv", "--agent", "--sandbox-deny-network", "--help", "-h", "--version", "-v":
		return true
	default:
		return false
	}
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
