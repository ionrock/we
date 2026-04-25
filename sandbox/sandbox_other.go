//go:build !darwin && !linux

package sandbox

import "fmt"

func newPlatformSandbox(cfg Config) (Sandbox, error) {
	return nil, fmt.Errorf("--agent sandbox is not supported on this platform (requires macOS or Linux)")
}
