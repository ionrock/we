package toconfig

import (
	"fmt"
	"io"
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

func ApplyConfig(t string, fh io.Writer) error {
	t, err := filepath.Abs(t)
	if err != nil {
		return err
	}
	name := filepath.Base(t)

	tmpl, err := template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(t)
	if err != nil {
		return err
	}
	err = tmpl.Execute(fh, envMap())
	if err != nil {
		return err
	}
	return nil
}

func parseTemplatePath(tmpl string) (string, string, error) {
	var target string

	if strings.Contains(tmpl, ":") {
		parts := strings.Split(tmpl, ":")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("template string must only have a template and target path")
		}
		tmpl = parts[0]
		target = parts[1]
	} else {
		target = strings.TrimRight(tmpl, ".tmpl")
	}

	return tmpl, target, nil
}

func ApplyTemplates(tmpls []string) error {
	for _, tmpl := range tmpls {
		tmpl, target, err := parseTemplatePath(tmpl)
		if err != nil {
			return err
		}

		fh := os.Stdout
		if target != "" {
			fh, err := os.Create(target)
			if err != nil {
				return err
			}
			defer fh.Close()
		}

		err = ApplyConfig(tmpl, fh)
		if err != nil {
			return err
		}
	}
	return nil
}
