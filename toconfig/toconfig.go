package toconfig

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
)

type ConfigTmpl struct {
	Template string `json:"template"`
	Target   string `json:"target"`
	Owner    string `json:"owner"`
	Group    string `json:"group"`
	FileMode string `json:"mode"`
}

func (conf *ConfigTmpl) Execute() error {
	fh := os.Stdout
	if conf.Target != "" {
		fh, err := os.Create(conf.Target)
		if err != nil {
			return err
		}
		defer fh.Close()
	}

	err := ApplyConfig(conf.Template, fh)
	if err != nil {
		return err
	}

	err = conf.SetPermissions()
	if err != nil {
		return err
	}

	return nil
}

func (conf *ConfigTmpl) SetPermissions() error {

	var uid int
	var gid int

	// Set some defaults based on the current user
	u, err := user.Current()
	if err != nil {
		return err
	}

	g, err := user.LookupGroupId(u.Gid)
	if err != nil {
		return err
	}

	// If we have an owner, update the user var
	if conf.Owner != "" {
		u, err = user.Lookup(conf.Owner)
		if err != nil {
			return err
		}
	}

	// if we have a group, update the group var
	if conf.Group != "" {
		g, err = user.LookupGroup(conf.Group)
		if err != nil {
			return nil
		}
	}

	// set the user and group id vars
	uid, err = strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}

	gid, err = strconv.Atoi(g.Gid)
	if err != nil {
		return nil
	}

	// chown the file
	err = os.Chown(conf.Target, uid, gid)
	if err != nil {
		return err
	}

	if conf.FileMode != "" {
		mode, err := strconv.ParseUint(conf.FileMode, 0, 32)
		if err != nil {
			return err
		}

		os.Chmod(conf.Target, os.FileMode(mode))
	}

	return nil
}

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

func parseTemplatePath(tmpl string) (*ConfigTmpl, error) {
	var target string

	if strings.Contains(tmpl, ":") {
		parts := strings.Split(tmpl, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("template string must only have a template and target path")
		}
		tmpl = parts[0]
		target = parts[1]
	} else {
		target = strings.TrimRight(tmpl, ".tmpl")
	}

	return &ConfigTmpl{
		Template: tmpl,
		Target:   target,
	}, nil
}

func ApplyTemplates(tmpls []string) error {
	for _, tmpl := range tmpls {
		conf, err := parseTemplatePath(tmpl)
		if err != nil {
			return err
		}

		err = conf.Execute()
		if err != nil {
			return err
		}
	}
	return nil
}
