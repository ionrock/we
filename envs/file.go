package envs

import (
	"os"

	"github.com/ionrock/we/flat"
	log "github.com/sirupsen/logrus"
)

type File struct {
	path string
}

func (e File) Parse() (map[string]string, error) {
	env, err := flat.NewFlatEnv(e.path)
	if err != nil {
		log.Fatal(err)
	}

	return env, nil
}

func (e File) Apply() (map[string]string, error) {
	env, err := e.Parse()
	if err != nil {
		return nil, err
	}

	for k, v := range env {
		log.Debugf("Setting: %s to %s", k, os.ExpandEnv(v))
		err = os.Setenv(k, os.ExpandEnv(v))
		if err != nil {
			return nil, err
		}
	}

	return env, nil
}
