//go:build darwin

package sandbox

import (
	"fmt"
	"strings"
)

// DarwinSandbox implements Sandbox using macOS sandbox-exec with SBPL profiles.
type DarwinSandbox struct {
	config Config
}

func newPlatformSandbox(cfg Config) (Sandbox, error) {
	return &DarwinSandbox{config: cfg}, nil
}

// Wrap returns a sandbox-exec invocation that applies the SBPL profile.
func (s *DarwinSandbox) Wrap(cmd string, args []string) (string, []string, error) {
	profile := s.buildProfile()

	sandboxArgs := []string{"-p", profile, cmd}
	sandboxArgs = append(sandboxArgs, args...)

	return "/usr/bin/sandbox-exec", sandboxArgs, nil
}

// buildProfile generates the SBPL (Seatbelt Profile Language) string.
// Uses (allow default) as the base with selective denies for specific paths.
func (s *DarwinSandbox) buildProfile() string {
	var b strings.Builder

	b.WriteString("(version 1)\n")
	b.WriteString("(allow default)\n")

	// Deny rules
	for _, rule := range s.config.Deny {
		if rule.Dir {
			fmt.Fprintf(&b, "(deny file-read* (subpath %q))\n", rule.Path)
		} else {
			fmt.Fprintf(&b, "(deny file-read* (literal %q))\n", rule.Path)
		}
	}

	// Allow overrides (must come after denies -- last matching rule wins)
	for _, rule := range s.config.Allow {
		if rule.Dir {
			fmt.Fprintf(&b, "(allow file-read* (subpath %q))\n", rule.Path)
		} else {
			fmt.Fprintf(&b, "(allow file-read* (literal %q))\n", rule.Path)
		}
	}

	// Optional network restriction
	if s.config.DenyNet {
		b.WriteString("(deny network*)\n")
	}

	return b.String()
}
