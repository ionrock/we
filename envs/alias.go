package envs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

func fileLocalPath(envPath string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	abspath, err := filepath.Abs(filepath.Dir(envPath))
	if err != nil {
		log.Error().Err(err).Msg("Error making file path absolute to env file")
		return path
	}
	return filepath.Join(abspath, path)
}

type Alias struct {
	path string
}

func (alias Alias) loadEntry(k, v string) (string, string) {
	if k == "file" {
		k = "env"
	}

	if k != "script" && k != "envvar" {
		v = fileLocalPath(alias.path, v)
	}

	return fmt.Sprintf("--%s", k), v
}

func (alias Alias) ApplyFromMap(entries []map[string]string, cur Env) (Env, error) {
	args := []string{}

	for _, e := range entries {
		for k, v := range e {
			flag, arg := alias.loadEntry(k, v)
			args = append(args, flag, arg)
		}
	}

	log.Debug().Msgf("Loaded alias %s with: %v", alias.path, args)

	env, err := WithEnvTypedFrom(args, filepath.Dir(alias.path), cur)
	if err != nil {
		return nil, err
	}
	return make(Env).Merge(cur).Merge(env), nil
}

func (alias Alias) Apply(cur Env) (Env, error) {
	log.Debug().Msgf("Reading: %s", alias.path)
	b, err := os.ReadFile(alias.path)
	if err != nil {
		return nil, err
	}

	// Use flexible unmarshaling to handle non-string entries (e.g., sandbox:)
	var flexEntries []map[string]interface{}
	if err := yaml.Unmarshal(b, &flexEntries); err != nil {
		return nil, err
	}

	// Filter to only string-valued entries
	entries := make([]map[string]string, 0)
	for _, entry := range flexEntries {
		strEntry := make(map[string]string)
		for k, v := range entry {
			if sv, ok := v.(string); ok {
				strEntry[k] = sv
			}
		}
		if len(strEntry) > 0 {
			entries = append(entries, strEntry)
		}
	}

	env, err := alias.ApplyFromMap(entries, cur)
	if err != nil {
		return nil, err
	}

	return env, nil
}
