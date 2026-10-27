package toconfig

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type ConfigTmpl struct {
	Template string `json:"template"`
	Target   string `json:"target"`
	Owner    string `json:"owner"`
	Group    string `json:"group"`
	FileMode string `json:"mode"`
}

// defaultFileMode is used for newly rendered files when no mode is given.
// Rendered templates receive the full environment, which commonly includes
// secrets, so new files are private by default.
const defaultFileMode os.FileMode = 0600

func (conf *ConfigTmpl) Execute() error {
	tmplAbs, err := filepath.Abs(conf.Template)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(conf.Target)
	if err != nil {
		return err
	}
	if tmplAbs == targetAbs {
		return fmt.Errorf("template %s would overwrite itself; use TEMPLATE:TARGET or a .tmpl suffix", conf.Template)
	}

	// Render fully before touching the target so a failing template never
	// truncates an existing file.
	var buf bytes.Buffer
	if err := ApplyConfig(conf.Template, &buf); err != nil {
		return err
	}

	mode, err := conf.fileMode(targetAbs)
	if err != nil {
		return err
	}

	// Write to a private temp file in the target directory, fix up
	// permissions and ownership, then atomically rename into place. This
	// avoids a window where secrets are readable with umask-derived
	// permissions and never follows a symlink planted at the target path.
	tmp, err := os.CreateTemp(filepath.Dir(targetAbs), "."+filepath.Base(targetAbs)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := conf.setOwnership(tmpName); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}

	if err := os.Rename(tmpName, targetAbs); err != nil {
		return err
	}
	committed = true
	return nil
}

// fileMode returns the configured mode, the existing target's mode, or the
// private default for new files.
func (conf *ConfigTmpl) fileMode(target string) (os.FileMode, error) {
	if conf.FileMode != "" {
		mode, err := strconv.ParseUint(conf.FileMode, 0, 32)
		if err != nil {
			return 0, err
		}
		return os.FileMode(mode).Perm(), nil
	}
	if info, err := os.Lstat(target); err == nil && info.Mode().IsRegular() {
		return info.Mode().Perm(), nil
	}
	return defaultFileMode, nil
}

// SetPermissions applies the configured ownership and mode to the target.
func (conf *ConfigTmpl) SetPermissions() error {
	if err := conf.setOwnership(conf.Target); err != nil {
		return err
	}
	if conf.FileMode != "" {
		mode, err := conf.fileMode(conf.Target)
		if err != nil {
			return err
		}
		return os.Chmod(conf.Target, mode)
	}
	return nil
}

func (conf *ConfigTmpl) setOwnership(path string) error {
	if conf.Owner == "" && conf.Group == "" {
		return nil
	}

	uid, gid := -1, -1

	if conf.Owner != "" {
		u, err := user.Lookup(conf.Owner)
		if err != nil {
			return err
		}
		uid, err = strconv.Atoi(u.Uid)
		if err != nil {
			return err
		}
	}

	if conf.Group != "" {
		g, err := user.LookupGroup(conf.Group)
		if err != nil {
			return err
		}
		gid, err = strconv.Atoi(g.Gid)
		if err != nil {
			return err
		}
	}

	return os.Lchown(path, uid, gid)
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
		target = strings.TrimSuffix(tmpl, ".tmpl")
	}

	return &ConfigTmpl{
		Template: tmpl,
		Target:   target,
	}, nil
}

func ApplyTemplates(tmpls []string) error {
	for _, tmpl := range tmpls {
		err := ApplyTemplate(tmpl)
		if err != nil {
			return err
		}
	}
	return nil
}

func ApplyTemplate(tmpl string) error {
	conf, err := parseTemplatePath(tmpl)
	if err != nil {
		return err
	}

	err = conf.Execute()
	if err != nil {
		return err
	}

	return nil
}
