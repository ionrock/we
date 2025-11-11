package envscript

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

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

// ParseEnvScript parses a shell script file containing export statements
func ParseEnvScript(path string) (map[string]string, error) {
	file, err := os.Open(path)
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
