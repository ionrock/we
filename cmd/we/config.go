package main

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

const (
	configFilename = ".withenv.yml"
)

func findConfig(curdir string) (string, error) {
	root, err := filepath.Abs("/")
	if err != nil {
		return "", err
	}

	curdir, err = filepath.Abs(curdir)
	if err != nil {
		return "", nil
	}

	log.Debug().Msgf("abs path: %q", curdir)

	config := filepath.Join(curdir, configFilename)
	log.Debug().Msgf("in findConfig: %q", config)
	if _, err := os.Stat(config); os.IsNotExist(err) {
		if curdir == root {
			return "", err
		} else {
			return findConfig(filepath.Dir(curdir))
		}
	}

	return filepath.Abs(config)
}
