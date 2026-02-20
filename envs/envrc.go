package envs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ionrock/we/envscript"
	"github.com/rs/zerolog/log"
)

const envrcFilename = ".envrc"

type Envrc struct {
	path string
}

func envBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func findEnvrc(curdir string) (string, error) {
	root, err := filepath.Abs("/")
	if err != nil {
		return "", err
	}

	curdir, err = filepath.Abs(curdir)
	if err != nil {
		return "", err
	}

	envrc := filepath.Join(curdir, envrcFilename)
	if _, err := os.Stat(envrc); os.IsNotExist(err) {
		if curdir == root {
			return "", os.ErrNotExist
		}
		return findEnvrc(filepath.Dir(curdir))
	}

	return filepath.Abs(envrc)
}

func (e Envrc) Parse() (map[string]string, error) {
	return envscript.ParseEnvScript(e.path)
}

func (e Envrc) Apply() (map[string]string, error) {
	env, err := e.Parse()
	if err != nil {
		return nil, err
	}

	for k, v := range env {
		expanded := os.ExpandEnv(v)
		log.Debug().Msgf("Setting from .envrc: %s=%s", k, expanded)
		if err := os.Setenv(k, expanded); err != nil {
			return nil, err
		}
	}

	return env, nil
}

func MaybeLoadEnvrc(startDir string, disabled bool) (map[string]string, error) {
	if disabled || envBool(os.Getenv("WE_NO_DIRENV")) {
		log.Debug().Msg("Skipping .envrc load: disabled by flag or WE_NO_DIRENV")
		return map[string]string{}, nil
	}

	envrcPath, err := findEnvrc(startDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debug().Msg("Skipping .envrc load: no .envrc found")
			return map[string]string{}, nil
		}
		return nil, err
	}

	log.Debug().Msgf("Loading .envrc from %s", envrcPath)
	return Envrc{path: envrcPath}.Apply()
}
