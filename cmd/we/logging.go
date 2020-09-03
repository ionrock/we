package main

import (
	log "github.com/sirupsen/logrus"
)

func InitLogging(debug bool) {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	if debug == true {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.Debug("Logging debug configured")
}
