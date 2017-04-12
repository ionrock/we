package envs

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

func fileLocalPath(envPath string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	abspath, err := filepath.Abs(filepath.Dir(envPath))
	if err != nil {
		log.Error("Error making file path absolute to env file", err)
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

func (alias Alias) ApplyFromMap(entries []map[string]string) (map[string]string, error) {
	args := []string{}

	for _, e := range entries {
		for k, v := range e {
			flag, arg := alias.loadEntry(k, v)
			args = append(args, flag, arg)
		}
	}

	log.Debugf("Loaded alias %s with: %s", alias.path, args)

	return WithEnv(args, filepath.Dir(alias.path))
}

func (alias Alias) Apply() (map[string]string, error) {
	log.Debug("Reading: ", alias.path)
	b, err := ioutil.ReadFile(alias.path)
	if err != nil {
		return nil, err
	}

	entries := make([]map[string]string, 0)

	yaml.Unmarshal(b, &entries)

	env, err := alias.ApplyFromMap(entries)
	if err != nil {
		return nil, err
	}

	return env, nil
}
