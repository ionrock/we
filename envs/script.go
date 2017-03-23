package envs

import (
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/ionrock/we"
)

type Script struct {
	cmd string
}

func (e Script) Apply() map[string]string {
	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmp.Name())

	parts := we.SplitCommand(e.cmd)
	for i := range parts {
		parts[i] = os.ExpandEnv(parts[i])
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = tmp
	err = cmd.Run()

	if err != nil {
		panic(err)
	}

	tmp.Close()

	ef := File{path: tmp.Name()}
	return ef.Apply()
}
