package main

import (
	"time"
)

type Watch struct {
	Command string
	Action  string
	Cadence uint64
}

func (w *Watch) Start() {
	ticker := time.NewTicker(time.Duration(w.Cadence))
	cmd := SplitCommand(w.Command)
	action := SplitCommand(w.Action)

	for range ticker.C {
		err := RunLogged(cmd...)
		if err != nil {
			RunLogged(action...)
		}
	}
}
