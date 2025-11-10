package envs

import (
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

func (e File) Apply() (map[string]string, error) {
	env, err := e.Parse()
	if err != nil {
		return nil, err
	}

	for k, v := range env {
		log.Debug().Msgf("Setting: %s to %s", k, os.ExpandEnv(v))
		err = os.Setenv(k, os.ExpandEnv(v))
		if err != nil {
			return nil, err
		}
	}

	return env, nil
}
