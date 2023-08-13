package main

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
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

	log.Debugf("abs path: %q", curdir)

	config := filepath.Join(curdir, configFilename)
	log.Debugf("in findConfig: %q", config)
	if _, err := os.Stat(config); os.IsNotExist(err) {
		if curdir == root {
			return "", err
		} else {
			return findConfig(filepath.Dir(curdir))
		}
	}

	return filepath.Abs(config)
}
