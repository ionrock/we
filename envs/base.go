package envs

import (
	log "github.com/Sirupsen/logrus"
)

type Action interface {
	Apply() map[string]string
}

func updateEnvMap(cur, env map[string]string) map[string]string {
	for k, v := range env {
		cur[k] = v
	}
	return cur
}

func ignore(flag string) bool {
	ignored := make(map[string]bool)

	ignored["--debug"] = true
	ignored["-D"] = true
	ignored["--clean"] = true
	ignored["-c"] = true

	_, ok := ignored[flag]
	return ok
}

func pairs(args []string) chan Action {
	p := make(chan Action)

	go func() {
		var flag string
		var action Action

		for _, f := range args {
			if flag == "" {
				if ignore(f) {
					continue
				} else {
					flag = f
				}
			} else {
				switch {
				case flag == "--env" || flag == "-e":
					action = File{path: f}
				case flag == "--script" || flag == "-s":
					action = Script{cmd: f}
				case flag == "--envvar" || flag == "-E":
					action = Var{field: f}
				case flag == "--directory" || flag == "-d":
					action = Dir{path: f}
				case flag == "--alias" || flag == "-a":
					action = Alias{path: f}
				default:
					action = nil
				}

				if action == nil {
					close(p)
					return
				} else {
					p <- action
					flag = ""
				}
			}
		}
		close(p)
	}()

	return p
}

func WithEnv(args []string) (map[string]string, error) {
	env := make(map[string]string)

	for action := range pairs(args) {
		log.Debugf("Applying action: %#v", action)
		env = updateEnvMap(env, action.Apply())
	}

	return env, nil
}
