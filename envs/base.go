package envs

import (
	"github.com/rs/zerolog/log"
)

type Action interface {
	Apply() (map[string]string, error)
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
	ignored["--no-direnv"] = true

	_, ok := ignored[flag]
	return ok
}

func pairs(args []string, path string) chan Action {
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
					action = Script{cmd: f, dir: path}
				case flag == "--envvar" || flag == "-E":
					action = Var{field: f, dir: path}
				case flag == "--directory" || flag == "-d":
					action = Dir{path: f}
				case flag == "--alias" || flag == "-a":
					action = Alias{path: f}
				case flag == "--template" || flag == "-t":
					action = Template{config: f}
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

func WithEnv(args []string, path string) (map[string]string, error) {
	env := make(map[string]string)

	for action := range pairs(args, path) {
		log.Debug().Msgf("Applying action: %#v", action)
		newEnv, err := action.Apply()
		if err != nil {
			return nil, err
		}
		env = updateEnvMap(env, newEnv)
	}

	return env, nil
}
