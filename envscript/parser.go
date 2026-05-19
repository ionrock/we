package envscript

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	shlex "github.com/flynn/go-shlex"
	"gopkg.in/yaml.v2"
)

// EnvVar represents a single environment variable
type EnvVar struct {
	Name  string
	Value string
}

var (
	// Match export statements like: export VAR=value or export VAR="value"
	exportPattern = regexp.MustCompile(`^\s*export\s+([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)
	// Match simple assignment like: VAR=value
	assignPattern = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)
	// Match comments
	commentPattern = regexp.MustCompile(`^\s*#`)
)

// ParseEnvScript parses a dotenv-style file containing export statements and
// simple direnv stdlib loading directives. It intentionally does not execute
// arbitrary shell code.
func ParseEnvScript(path string) (map[string]string, error) {
	return parseEnvScript(path, map[string]bool{})
}

func parseEnvScript(path string, seen map[string]bool) (map[string]string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	if seen[absPath] {
		return map[string]string{}, nil
	}
	seen[absPath] = true

	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if strings.TrimSpace(line) == "" || commentPattern.MatchString(line) {
			continue
		}

		if loadedEnv, ok, err := parseDirenvLoad(line, filepath.Dir(absPath), seen); ok {
			if err != nil {
				return nil, err
			}
			for key, value := range loadedEnv {
				env[key] = value
			}
			continue
		}

		// Try to match export statements first
		if matches := exportPattern.FindStringSubmatch(line); matches != nil {
			name := matches[1]
			value := cleanValue(matches[2])
			env[name] = value
			continue
		}

		// Try to match simple assignments
		if matches := assignPattern.FindStringSubmatch(line); matches != nil {
			name := matches[1]
			value := cleanValue(matches[2])
			env[name] = value
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return env, nil
}

func parseDirenvLoad(line string, baseDir string, seen map[string]bool) (map[string]string, bool, error) {
	parts, err := shlex.Split(line)
	if err != nil || len(parts) == 0 {
		return nil, false, nil
	}

	cmd := parts[0]
	if cmd != "dotenv" && cmd != "dotenv_if_exists" && cmd != "source_env" && cmd != "source_env_if_exists" {
		return nil, false, nil
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
			return map[string]string{}, true, nil
		}
		return nil, true, fmt.Errorf("%s %s: %w", cmd, target, err)
	}

	loadedEnv, err := parseEnvScript(target, seen)
	if err != nil {
		return nil, true, err
	}
	return loadedEnv, true, nil
}

// cleanValue removes surrounding quotes and handles escaping
func cleanValue(value string) string {
	value = strings.TrimSpace(value)

	// Remove trailing comments (only if not inside quotes)
	if idx := strings.Index(value, "#"); idx > 0 {
		// Check if the # is outside quotes
		inQuotes := false
		quoteChar := rune(0)
		for i, ch := range value {
			if ch == '"' || ch == '\'' {
				if !inQuotes {
					inQuotes = true
					quoteChar = ch
				} else if ch == quoteChar {
					inQuotes = false
					quoteChar = 0
				}
			}
			if ch == '#' && !inQuotes {
				value = strings.TrimSpace(value[:i])
				break
			}
		}
	}

	// Remove surrounding quotes
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	// Handle escaped quotes and other escape sequences
	value = strings.ReplaceAll(value, `\"`, `"`)
	value = strings.ReplaceAll(value, `\'`, `'`)
	value = strings.ReplaceAll(value, `\\`, `\`)

	return value
}

// ConvertToYAML converts environment variables map to YAML list format
func ConvertToYAML(env map[string]string) ([]byte, error) {
	// Create a list of maps, each containing a single key-value pair
	var envList []map[string]string

	// Sort keys for consistent output
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}

	// Sort keys alphabetically
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	// Build list of single-key maps
	for _, key := range keys {
		envList = append(envList, map[string]string{key: env[key]})
	}

	yamlData, err := yaml.Marshal(envList)
	if err != nil {
		return nil, err
	}

	// Prepend YAML document separator
	result := append([]byte("---\n"), yamlData...)
	return result, nil
}

// ParseAndConvert is a convenience function that parses and converts in one step
func ParseAndConvert(path string) ([]byte, error) {
	env, err := ParseEnvScript(path)
	if err != nil {
		return nil, err
	}

	return ConvertToYAML(env)
}
