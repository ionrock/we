package envs

import (
	"github.com/rs/zerolog/log"
)

type Action interface {
	Apply(cur Env) (Env, error)
}

func ignore(flag string) bool {
	ignored := make(map[string]bool)

	ignored["--debug"] = true
	ignored["-D"] = true
	ignored["--clean"] = true
	ignored["-c"] = true
	ignored["--no-direnv"] = true
	ignored["--agent"] = true
	ignored["--sandbox-deny-network"] = true
	ignored["--show-secrets"] = true

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
				case flag == "--sandbox-deny" || flag == "--sandbox-allow":
					// Skip sandbox flags -- handled by WeAction
					flag = ""
					continue
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

func WithEnvTyped(args []string, path string) (Env, error) {
	return WithEnvTypedFrom(args, path, nil)
}

func WithEnvTypedFrom(args []string, path string, initial Env) (Env, error) {
	env := make(Env)
	if initial != nil {
		env = env.Merge(initial)
	}

	for action := range pairs(args, path) {
		log.Debug().Msgf("Applying action: %#v", action)
		newEnv, err := action.Apply(env)
		if err != nil {
			return nil, err
		}
		env = env.Merge(newEnv)
	}

	return env, nil
}

func WithEnv(args []string, path string) (map[string]string, error) {
	env, err := WithEnvTyped(args, path)
	if err != nil {
		return nil, err
	}
	return env.StringMap(), nil
}
