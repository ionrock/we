package sandbox

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runHook(t *testing.T, script, command string) []byte {
	t.Helper()
	input, _ := json.Marshal(map[string]interface{}{
		"tool_input": map[string]string{"command": command},
	})
	cmd := exec.Command("bash", script)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("hook failed for %q: %v", command, err)
	}
	return out
}

func TestHookScriptQuotesConfigValues(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not installed")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not installed")
	}

	dir := t.TempDir()
	marker := filepath.Join(dir, "pwned")
	script := filepath.Join(dir, "hook.sh")

	cfg := HooksConfig{
		WeBinary: filepath.Join(dir, "bin dir", "we$(touch "+marker+")"),
		SkipPrefix: []string{
			"x ]]; touch " + marker + "; [[ x",
			"*",
			"make",
		},
	}
	if err := writeHookScript(script, cfg); err != nil {
		t.Fatal(err)
	}

	out := runHook(t, script, "go test ./...")
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("config value was executed by the hook script")
	}

	var resp struct {
		HookSpecificOutput map[string]interface{} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("hook output is not JSON: %v: %s", err, out)
	}
	if _, ok := resp.HookSpecificOutput["permissionDecision"]; ok {
		t.Error("hook must not set permissionDecision (would auto-approve commands)")
	}
	updated, _ := resp.HookSpecificOutput["updatedInput"].(map[string]interface{})
	got, _ := updated["command"].(string)
	want := shellQuote(cfg.WeBinary) + " go test ./..."
	if got != want {
		t.Errorf("rewritten command = %q, want %q", got, want)
	}

	// "*" must be a literal prefix, not a glob that skips every command.
	if out := runHook(t, script, "make build"); len(bytes.TrimSpace(out)) != 0 {
		t.Errorf("skip-prefix command should pass through, got %s", out)
	}
	if out := runHook(t, script, "makefoo"); !strings.Contains(string(out), "updatedInput") {
		t.Errorf("skip-prefix must match whole words only, got %s", out)
	}
	if out := runHook(t, script, want); len(bytes.TrimSpace(out)) != 0 {
		t.Errorf("already-prefixed command should pass through, got %s", out)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("config value was executed by the hook script")
	}
}
