package process

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	shlex "github.com/flynn/go-shlex"
	log "github.com/sirupsen/logrus"
)

// try to find the exit code of an error. If it is unavailable, it
// will use 1
func exitCode(err error) int {
	// https://stackoverflow.com/questions/10385551/get-exit-code-go

	// try to get the exit code
	if exitError, ok := err.(*exec.ExitError); ok {
		ws := exitError.Sys().(syscall.WaitStatus)
		return ws.ExitStatus()
	}
	// This will happen (in OSX) if `name` is not available in $PATH,
	// in this situation, exit code could not be get, and stderr will be
	// empty string very likely, so we use the default fail code, and format err
	// to string and set to stderr
	return 1
}

// RunAndWait a helper to run a command and wait for it to finish
// before returning the status code as an int.
func RunAndWait(cmd *exec.Cmd) (int, error) {
	err := cmd.Run()
	if err != nil {
		return exitCode(err), err
	}

	return 0, err
}

func SplitCommand(cmd string) []string {
	parts, err := shlex.Split(os.ExpandEnv(cmd))
	if err != nil {
		log.Fatal(err)
	}
	return parts
}

func procDir(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	f, err := os.Stat(abs)
	if err != nil {
		return "", err
	}

	if f.IsDir() {
		return abs, nil
	}

	return filepath.Dir(abs), nil
}

func CompileValue(value string, path string) (string, error) {
	log.Debugf("%#v", value)
	if strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`") {
		dirname, err := procDir(path)
		if err != nil {
			return "", err
		}

		proc := Process{
			Cmd: strings.Trim(value, "`"),
			Dir: dirname,
		}
		buf, err := proc.Execute()
		if err != nil {
			return "", err
		}

		return string(bytes.TrimSpace(buf.Bytes())), nil
	}
	return value, nil
}

type Process struct {
	Cmd string
	Dir string
}

func New(cmd string, dir string) *Process {
	return &Process{cmd, dir}
}

func (p *Process) Execute() (*bytes.Buffer, error) {
	cmds := findCmds(p.Cmd)
	if p.Dir != "" {
		for i := range cmds {
			cmds[i].Dir = p.Dir
		}
	}

	var buf bytes.Buffer
	err := execute(&buf, cmds...)
	if err != nil {
		return nil, err
	}

	return &buf, nil
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

func execute(output_buffer *bytes.Buffer, stack ...*exec.Cmd) error {
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
	parts := SplitCommand(cmdstr)
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
