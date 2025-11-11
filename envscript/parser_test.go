package envscript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvScript(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.env")

	content := `# Test environment file
export DATABASE_URL="postgresql://localhost:5432/mydb"
export API_KEY=abc123
export DEBUG=true

# Application settings
export APP_NAME="My Application"
export PORT=3000
HOST='localhost'

# Empty value
export EMPTY_VAR=

# Inline comment
export WITH_COMMENT=value # this is a comment
export QUOTED_COMMENT="value # not a comment"
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	env, err := ParseEnvScript(testFile)
	if err != nil {
		t.Fatalf("ParseEnvScript failed: %v", err)
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"DATABASE_URL", "postgresql://localhost:5432/mydb"},
		{"API_KEY", "abc123"},
		{"DEBUG", "true"},
		{"APP_NAME", "My Application"},
		{"PORT", "3000"},
		{"HOST", "localhost"},
		{"EMPTY_VAR", ""},
		{"WITH_COMMENT", "value"},
		{"QUOTED_COMMENT", "value # not a comment"},
	}

	for _, tt := range tests {
		if got, ok := env[tt.key]; !ok {
			t.Errorf("Expected key %q to be present", tt.key)
		} else if got != tt.expected {
			t.Errorf("For key %q: expected %q, got %q", tt.key, tt.expected, got)
		}
	}
}

func TestCleanValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`'world'`, "world"},
		{`value`, "value"},
		{`"value with spaces"`, "value with spaces"},
		{`value # comment`, "value"},
		{`"value # not comment"`, "value # not comment"},
		{`'value # not comment'`, "value # not comment"},
		{`"escaped \"quotes\""`, `escaped "quotes"`},
		{`  spaced  `, "spaced"},
	}

	for _, tt := range tests {
		got := cleanValue(tt.input)
		if got != tt.expected {
			t.Errorf("cleanValue(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestConvertToYAML(t *testing.T) {
	env := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value with spaces",
	}

	yaml, err := ConvertToYAML(env)
	if err != nil {
		t.Fatalf("ConvertToYAML failed: %v", err)
	}

	// Just check that we got some YAML output
	if len(yaml) == 0 {
		t.Error("Expected non-empty YAML output")
	}

	yamlStr := string(yaml)

	// Check for YAML document separator
	if !contains(yamlStr, "---\n") {
		t.Error("Expected YAML document separator '---' at the beginning")
	}

	// Check that keys are present in output as list items
	if !contains(yamlStr, "- KEY1:") || !contains(yamlStr, "- KEY2:") || !contains(yamlStr, "- KEY3:") {
		t.Error("Expected all keys to be present in YAML output as list items")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
