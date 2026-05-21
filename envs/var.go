package envs

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/ionrock/we/flat"
	"github.com/ionrock/we/process"
)

type Var struct {
	field string
	dir   string
}

func (e Var) Apply(cur Env) (Env, error) {
	parts := strings.SplitN(e.field, "=", 2)
	if len(parts) != 2 {
		return nil, errors.New("Invalid env var format. Use %s=%s")
	}
	key := parts[0]
	raw := parts[1]

	raw, err := process.CompileValue(raw, e.dir)
	if err != nil {
		return nil, err
	}

	// Preserve existing side effects for command substitution/expansion users.
	legacy := make(map[string]string)
	flat.ApplyString(legacy, key, raw)

	expanded, usedSecret := expandValue(raw, cur)
	env := make(Env)
	if ref, optional, ok := isSecretRef(expanded); ok {
		value, include, err := DefaultProviders().Resolve(context.Background(), ref, optional)
		if err != nil {
			return nil, err
		}
		if include {
			value.Raw = raw
			value.Source = "envvar"
			env[key] = value
			os.Setenv(key, value.Value)
		}
		return env, nil
	}
	value := NewLiteralValue(expanded, "envvar")
	value.Raw = raw
	value.Secret = usedSecret || LooksSensitiveName(key)
	env[key] = value
	os.Setenv(key, value.Value)
	return env, nil
}
