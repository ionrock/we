package envs

import (
	"errors"
	"strings"

	"github.com/ionrock/we/flat"
	"github.com/ionrock/we/process"
)

type Var struct {
	field string
	dir   string
}

func (e Var) Apply() (map[string]string, error) {
	parts := strings.Split(e.field, "=")
	if len(parts) != 2 {
		return nil, errors.New("Invalid env var format. Use %s=%s")
	}
	key := parts[0]
	value := parts[1]

	value, err := process.CompileValue(value, e.dir)
	if err != nil {
		return nil, err
	}

	env := make(map[string]string)
	flat.ApplyString(env, key, value)

	return env, nil
}
