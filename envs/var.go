package envs

import (
	"strings"

	"github.com/ionrock/we"

	log "github.com/Sirupsen/logrus"
)

type Var struct {
	field string
	dir   string
}

func (e Var) Apply() map[string]string {
	parts := strings.Split(e.field, "=")
	if len(parts) != 2 {
		log.Fatal("Invalid env var format. Use %s=%s")
	}
	key := parts[0]
	value := parts[1]
	value = we.CompileValue(value, e.dir)

	env := make(map[string]string)
	we.ApplyString(env, key, value)

	return env
}
