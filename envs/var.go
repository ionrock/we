package envs

import (
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type Var struct {
	field string
}

func (e Var) Apply() map[string]string {
	parts := strings.Split(e.field, "=")
	if len(parts) != 2 {
		log.Fatal("Invalid env var format. Use %s=%s")
	}
	key := parts[0]
	value := parts[1]

	env := make(map[string]string)
	env[key] = value

	err := os.Setenv(key, value)
	if err != nil {
		log.Fatal(err)
	}
	return env
}
