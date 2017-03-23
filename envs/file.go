package envs

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/ionrock/we"
)

type File struct {
	path string
}

func (e File) Parse() (map[string]string, error) {
	env, err := we.NewFlatEnv(e.path)
	if err != nil {
		log.Fatal(err)
	}

	return env, nil
}

func (e File) Apply() map[string]string {
	env, err := e.Parse()
	if err != nil {
		panic(err)
	}

	for k, v := range env {
		log.Debugf("Setting: %s to %s", k, os.ExpandEnv(v))
		err = os.Setenv(k, os.ExpandEnv(v))
		if err != nil {
			panic(err)
		}
	}

	return env
}
