package envs

import (
	"os"
	"path/filepath"
	"strings"
)

type Dir struct {
	path string
}

func (e Dir) Files() chan string {
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

func (e Dir) Apply() (map[string]string, error) {
	env := make(map[string]string)

	for fn := range e.Files() {
		ef := File{fn}
		newEnv, err := ef.Apply()
		if err != nil {
			return nil, err
		}
		env = updateEnvMap(env, newEnv)
	}

	return env, nil
}
