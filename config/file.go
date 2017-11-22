package config

import (
	"os"

	log "github.com/Sirupsen/logrus"

	"github.com/ionrock/we/flat"
	"github.com/ionrock/we/process"
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

func (e File) Apply(config *Config) error {
	env, err := e.Parse()
	if err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	for k, v := range env {
		log.Debugf("Setting: %s to %s", k, os.Expand(v, config.GetConfig))
		val, err := process.CompileValue(os.Expand(v, config.GetConfig), wd)
		if err != nil {
			return err
		}
		config.Set(k, val)
	}

	return nil
}
