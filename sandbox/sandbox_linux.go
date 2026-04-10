//go:build linux

package sandbox

import (
	"github.com/rs/zerolog/log"
)

// LinuxSandbox implements Sandbox as a stub that warns about missing Landlock support.
// TODO: Implement Landlock LSM support using github.com/landlock-lsm/go-landlock/landlock
type LinuxSandbox struct {
	config Config
}

func newPlatformSandbox(cfg Config) (Sandbox, error) {
	log.Warn().Msg("Linux sandbox (Landlock) not yet implemented; running without filesystem restrictions")
	return &LinuxSandbox{config: cfg}, nil
}

// Wrap returns the original command and args unmodified.
func (s *LinuxSandbox) Wrap(cmd string, args []string) (string, []string, error) {
	return cmd, args, nil
}
