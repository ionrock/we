package envs

import (
	"fmt"
	"os"

	"github.com/ionrock/we/toconfig"
)

type Template struct {
	config string
}

func (t Template) Apply() (map[string]string, error) {
	err := toconfig.ApplyTemplate(t.config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing template: %q", err)
		os.Exit(1)
	}

	return nil, nil
}
