package we

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
)

func envMap() map[string]string {
	result := map[string]string{}
	for _, envvar := range os.Environ() {
		parts := strings.SplitN(envvar, "=", 2)
		result[parts[0]] = parts[1]
	}
	return result
}

func ApplyConfig(t string, c string) error {

	t, err := filepath.Abs(t)
	if err != nil {
		panic(err)
	}
	name := filepath.Base(t)

	tmpl, err := template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(t)
	if err != nil {
		panic(err)
	}

	fh := os.Stdout
	if c != "" {
		fh, err = os.Create(c)
		if err != nil {
			panic(err)
		}
		defer fh.Close()
	}

	err = tmpl.Execute(fh, envMap())
	if err != nil {
		panic(err)
	}
	return nil
}
