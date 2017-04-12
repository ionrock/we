package envs

import (
	"io/ioutil"
	"os"

	"github.com/ionrock/we/process"
)

type Script struct {
	cmd string
	dir string
}

func (e Script) Apply() (map[string]string, error) {
	proc := process.New(e.cmd, e.dir)

	buf, err := proc.Execute()
	if err != nil {
		return nil, err
	}

	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())

	tmp.Write(buf.Bytes())
	tmp.Close()

	ef := File{path: tmp.Name()}
	return ef.Apply()
}
