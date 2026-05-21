package envs

import (
	"context"
	"os"

	"github.com/ionrock/we/flat"
	"github.com/rs/zerolog/log"
)

type File struct {
	path string
}

func (e File) Parse() (map[string]string, error) {
	env, err := flat.NewFlatEnv(e.path)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse flat environment")
	}

	return env, nil
}

func (e File) parseWithOrder() (map[string]string, []string, error) {
	env, order, err := flat.NewFlatEnvWithOrder(e.path)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse flat environment")
	}
	return env, order, nil
}

func (e File) Apply(cur Env) (Env, error) {
	parsed, order, err := e.parseWithOrder()
	if err != nil {
		return nil, err
	}

	env := make(Env, len(parsed))
	providers := DefaultProviders()
	working := make(Env).Merge(cur)
	for _, k := range order {
		raw := parsed[k]
		expanded, usedSecret := expandValue(raw, working)
		if ref, optional, ok := isSecretRef(expanded); ok {
			value, include, err := providers.Resolve(context.Background(), ref, optional)
			if err != nil {
				return nil, err
			}
			if !include {
				continue
			}
			value.Raw = raw
			value.Source = e.path
			env[k] = value
			working[k] = value
		} else {
			value := NewLiteralValue(expanded, e.path)
			value.Raw = raw
			value.Secret = usedSecret || LooksSensitiveName(k)
			env[k] = value
			working[k] = value
		}
		log.Debug().Msgf("Setting: %s to %s", k, env[k].DisplayForKey(k, false))
		if err = os.Setenv(k, env[k].Value); err != nil {
			return nil, err
		}
	}

	return env, nil
}
