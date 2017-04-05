package envs

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/ionrock/we"
)

type Script struct {
	cmd string
	dir string
}

func call(stack []*exec.Cmd, pipes []*io.PipeWriter) (err error) {
	if stack[0].Process == nil {
		if err = stack[0].Start(); err != nil {
			return err
		}
	}
	if len(stack) > 1 {
		if err = stack[1].Start(); err != nil {
			return err
		}
		defer func() {
			if err == nil {
				pipes[0].Close()
				err = call(stack[1:], pipes[1:])
			}
		}()
	}
	return stack[0].Wait()
}

func Execute(output_buffer *bytes.Buffer, stack ...*exec.Cmd) error {
	var errbuf bytes.Buffer
	pipe_stack := make([]*io.PipeWriter, len(stack)-1)

	i := 0
	for ; i < len(stack)-1; i++ {
		stdin_pipe, stdout_pipe := io.Pipe()
		stack[i].Stdout = stdout_pipe
		stack[i].Stderr = &errbuf

		// set the input to the outoput
		stack[i+1].Stdin = stdin_pipe
		pipe_stack[i] = stdout_pipe
	}
	stack[i].Stdout = output_buffer
	stack[i].Stderr = &errbuf

	if err := call(stack, pipe_stack); err != nil {
		log.Debug(string(errbuf.Bytes()))
		return err
	}
	return nil
}

func addCmd(cmds []*exec.Cmd, cmd []string) []*exec.Cmd {
	if len(cmd) == 1 {
		cmds = append(cmds, exec.Command(cmd[0]))
	} else {
		cmds = append(cmds, exec.Command(cmd[0], cmd[1:]...))
	}
	return cmds
}

func findCmds(cmdstr string) []*exec.Cmd {
	parts := we.SplitCommand(cmdstr)
	for i := range parts {
		parts[i] = os.ExpandEnv(parts[i])
	}

	cmds := []*exec.Cmd{}

	cmd := []string{}
	for _, p := range parts {
		if p == "|" {
			cmds = addCmd(cmds, cmd)
			cmd = []string{}
		} else {
			cmd = append(cmd, p)
		}
	}

	if len(cmd) > 0 {
		cmds = addCmd(cmds, cmd)
	}

	for _, c := range cmds {
		log.Debugf("Parsed cmd: %s", c)
	}
	return cmds
}

func (e Script) Apply() map[string]string {
	cmds := findCmds(e.cmd)
	if e.dir != "" {
		for i := range cmds {
			cmds[i].Dir = e.dir
		}
	}

	var buf bytes.Buffer
	err := Execute(&buf, cmds...)

	if err != nil {
		panic(err)
	}

	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmp.Name())

	tmp.Write(buf.Bytes())
	tmp.Close()

	ef := File{path: tmp.Name()}
	return ef.Apply()
}
