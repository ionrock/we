package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

// HooksConfig configures the Claude Code hook generation.
type HooksConfig struct {
	WeBinary   string   // Absolute path to the `we` binary
	SkipPrefix []string // Commands to not prefix with `we`
	ProjectDir string   // Project directory (where .claude/ lives)
}

// hookEntry matches Claude Code's settings.json hook schema.
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type hookMatcher struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

type hooksSection struct {
	PreToolUse []hookMatcher `json:"PreToolUse,omitempty"`
}

type settingsFile struct {
	Hooks *hooksSection `json:"hooks,omitempty"`
}

// WriteProjectSettings writes the PreToolUse hook into .claude/settings.local.json
// and creates the hook script in a temp directory. Returns a cleanup function
// that restores the original settings and removes temp files.
func WriteProjectSettings(cfg HooksConfig) (func(), error) {
	claudeDir := filepath.Join(cfg.ProjectDir, ".claude")
	settingsPath := filepath.Join(claudeDir, "settings.local.json")

	// Save original settings for restoration
	var originalContent []byte
	originalExists := false
	if data, err := os.ReadFile(settingsPath); err == nil {
		originalContent = data
		originalExists = true
	}

	// Create temp dir for hook script
	tmpDir, err := os.MkdirTemp("", "we-agent-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	// Write the hook script
	hookScript := filepath.Join(tmpDir, "hook.sh")
	if err := writeHookScript(hookScript, cfg); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("writing hook script: %w", err)
	}

	// Build our hook configuration
	ourHook := hookMatcher{
		Matcher: "Bash",
		Hooks: []hookEntry{
			{Type: "command", Command: hookScript},
		},
	}

	// Read and merge with existing settings
	var settings map[string]interface{}
	if originalExists {
		if err := json.Unmarshal(originalContent, &settings); err != nil {
			settings = make(map[string]interface{})
		}
	} else {
		settings = make(map[string]interface{})
	}

	// Merge our hook into the settings
	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
	}

	// Marshal our hook matcher to interface{} for merging
	hookBytes, _ := json.Marshal(ourHook)
	var hookInterface interface{}
	json.Unmarshal(hookBytes, &hookInterface)

	// Append to existing PreToolUse hooks
	existing, _ := hooks["PreToolUse"].([]interface{})
	existing = append(existing, hookInterface)
	hooks["PreToolUse"] = existing
	settings["hooks"] = hooks

	// Ensure .claude directory exists
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("creating .claude dir: %w", err)
	}

	// Write the merged settings
	merged, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("marshaling settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, merged, 0644); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("writing settings: %w", err)
	}

	log.Debug().Str("settings", settingsPath).Str("hook", hookScript).Msg("wrote agent hooks")

	// Return cleanup function (safe to call multiple times)
	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			os.RemoveAll(tmpDir)
			if originalExists {
				os.WriteFile(settingsPath, originalContent, 0644)
				log.Debug().Msg("restored original settings.local.json")
			} else {
				os.Remove(settingsPath)
				// Remove .claude dir if we created it and it's empty
				entries, _ := os.ReadDir(claudeDir)
				if len(entries) == 0 {
					os.Remove(claudeDir)
				}
				log.Debug().Msg("removed settings.local.json")
			}
		})
	}

	return cleanup, nil
}

// writeHookScript generates the bash hook script that prefixes Bash commands with `we`.
func writeHookScript(path string, cfg HooksConfig) error {
	weBin := cfg.WeBinary
	if weBin == "" {
		weBin = "we"
	}

	// The prefix as it should appear in the rewritten command line. The binary
	// path is shell-quoted so paths containing spaces or metacharacters are
	// passed as a single word rather than interpreted by the shell.
	wePrefix := shellQuote(weBin)

	// Commands (matched as a whole word at the start of the command) that must
	// not be prefixed. Shell builtins that change shell state are skipped.
	skip := []string{"cd", "export", "source", ".", "alias", "unset"}
	skip = append(skip, cfg.SkipPrefix...)

	// All dynamic values are embedded as single-quoted shell literals and
	// compared with quoted expansions, so config-provided values (for example
	// skip-prefix entries from .withenv.yml) cannot inject shell code into the
	// hook or act as glob patterns.
	quotedSkip := make([]string, 0, len(skip))
	for _, s := range skip {
		if s == "" {
			continue
		}
		quotedSkip = append(quotedSkip, shellQuote(s))
	}

	script := fmt.Sprintf(`#!/bin/bash
# Generated by we --agent. Prefixes Bash commands with we for env re-evaluation.
set -euo pipefail

WE_PREFIX=%s
SKIP_PREFIXES=(%s)

INPUT=$(cat)
COMMAND=$(printf '%%s' "$INPUT" | jq -r '.tool_input.command // empty')

# If we couldn't extract a command, pass through
if [[ -z "$COMMAND" ]]; then
  exit 0
fi

# Already prefixed with we
if [[ "$COMMAND" == "$WE_PREFIX" || "$COMMAND" == "$WE_PREFIX "* ]]; then
  exit 0
fi

# Skip commands that should not be prefixed
for prefix in "${SKIP_PREFIXES[@]}"; do
  if [[ "$COMMAND" == "$prefix" || "$COMMAND" == "$prefix "* ]]; then
    exit 0
  fi
done

# Prefix with we so env sources are re-evaluated. No permissionDecision is
# returned: the rewritten command still goes through Claude Code's normal
# permission evaluation instead of being auto-approved.
jq -n --arg cmd "$WE_PREFIX $COMMAND" '{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "updatedInput": { "command": $cmd }
  }
}'
`, shellQuote(wePrefix), strings.Join(quotedSkip, " "))

	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		return err
	}

	return nil
}

// shellQuote returns s as a single-quoted POSIX shell literal.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
