package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
)

func fileLocalPath(envPath string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	abspath, err := filepath.Abs(filepath.Dir(envPath))
	if err != nil {
		log.Error("Error making file path absolute to env file", err)
		return path
	}
	return filepath.Join(abspath, path)
}

type Action interface {
	Apply() map[string]string
}

type EnvVar struct {
	field string
}

func (e EnvVar) Apply() map[string]string {
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

type EnvFile struct {
	path string
}

func (e EnvFile) Parse() (map[string]string, error) {
	f := Flattener{e.path}

	env, err := f.Flatten()
	if err != nil {
		log.Fatal(err)
	}

	return env, nil
}

func (e EnvFile) Apply() map[string]string {
	env, err := e.Parse()
	if err != nil {
		panic(err)
	}

	for k, v := range env {
		log.Debugf("Setting: %s to %s", k, os.ExpandEnv(v))
		err = os.Setenv(k, os.ExpandEnv(v))
		if err != nil {
			panic(err)
		}
	}

	return env
}

type EnvDir struct {
	path string
}

func (e EnvDir) Files() chan string {
	files := make(chan string)

	go func() {
		extensions := []string{"yaml", "yml", "json"}

		filepath.Walk(e.path, func(path string, f os.FileInfo, err error) error {
			if !f.IsDir() {
				for _, ext := range extensions {
					if strings.HasSuffix(path, ext) {
						files <- path
					}
				}
			}
			return nil
		})

		close(files)
	}()

	return files
}

func (e EnvDir) Apply() map[string]string {
	env := make(map[string]string)

	for fn := range e.Files() {
		ef := EnvFile{fn}
		env = updateEnvMap(env, ef.Apply())
	}

	return env
}

type EnvScript struct {
	cmd string
}

func (e EnvScript) Apply() map[string]string {
	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmp.Name())

	parts := SplitCommand(e.cmd)

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = tmp
	err = cmd.Run()

	if err != nil {
		panic(err)
	}

	tmp.Close()

	ef := EnvFile{path: tmp.Name()}
	return ef.Apply()
}

type EnvAlias struct {
	path string
}

func (alias EnvAlias) ApplyFromMap(entries []map[string]string) (map[string]string, error) {
	args := []string{}

	for _, e := range entries {
		for k, v := range e {
			if k == "file" {
				args = append(args, "--env", fileLocalPath(alias.path, v))
			} else {
				args = append(args, fmt.Sprintf("--%s", k), fileLocalPath(alias.path, v))
			}
		}
	}

	log.Debugf("Loaded alias %s with: %s", alias.path, args)

	return WithEnv(args)
}

func (alias EnvAlias) Apply() map[string]string {
	log.Debug("Reading: ", alias.path)
	b, err := ioutil.ReadFile(alias.path)
	if err != nil {
		log.Fatal(err)
	}

	entries := make([]map[string]string, 0)

	yaml.Unmarshal(b, &entries)

	env, err := alias.ApplyFromMap(entries)
	if err != nil {
		log.Fatal(err)
	}

	return env
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
					action = EnvFile{path: f}
				case flag == "--script" || flag == "-s":
					action = EnvScript{cmd: f}
				case flag == "--envvar" || flag == "-E":
					action = EnvVar{field: f}
				case flag == "--directory" || flag == "-d":
					action = EnvDir{path: f}
				case flag == "--alias" || flag == "-a":
					action = EnvAlias{path: f}
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
		log.Debug("Applying action: ", action)
		env = updateEnvMap(env, action.Apply())
	}

	return env, nil
}
