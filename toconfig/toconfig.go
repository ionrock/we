package toconfig

import (
	"fmt"
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
		return err
	}
	name := filepath.Base(t)

	tmpl, err := template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(t)
	if err != nil {
		return err
	}

	fh := os.Stdout
	if c != "" {
		fh, err = os.Create(c)
		if err != nil {
			return err
		}
		defer fh.Close()
	}

	err = tmpl.Execute(fh, envMap())
	if err != nil {
		return err
	}
	return nil
}

func ApplyTemplates(tmpls []string) error {
	for _, tmpl := range tmpls {
		var target string

		if strings.Contains(tmpl, ":") {
			parts := strings.Split(tmpl, ":")
			if len(parts) != 2 {
				return fmt.Errorf("template string must only have a template and target path")
			}
			tmpl = parts[0]
			target = parts[1]

		} else {
			target = strings.TrimRight(tmpl, ".tmpl")
		}
		err := ApplyConfig(tmpl, target)
		if err != nil {
			return err
		}
	}
	return nil
}
