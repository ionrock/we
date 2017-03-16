package we

import (
	"bufio"
	"os"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/flynn/go-shlex"
	"github.com/urfave/cli"
)

func RunLogged(parts ...string) error {
	log.Debug("Running command: ", parts)
	cmd := exec.Command(parts[0], parts[1:]...)

	o, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("Error creating stdout pipe: ", err)
	}

	e, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal("Error creating stderr pipe: ", err)
	}

	stdout := bufio.NewScanner(o)
	stderr := bufio.NewScanner(e)
	go func() {
		for stdout.Scan() {
			log.Info(stdout.Text())
		}
	}()

	go func() {
		for stderr.Scan() {
			log.Info(stderr.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		log.Fatal("Error starting cmd: ", err)
	}

	return cmd.Wait()
}

func SplitCommand(cmd string) []string {
	parts, err := shlex.Split(os.ExpandEnv(cmd))
	if err != nil {
		log.Fatal(err)
	}
	return parts
}

func RunWrapped(parts ...string) error {
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func NewCommand(script string) *exec.Cmd {
	parts := SplitCommand(script)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd
}

func CommandAction(c *cli.Context) error {
	args := c.Args()

	if len(args) > 0 {
		parts := make([]string, len(args))

		for i, arg := range args {
			parts[i] = os.ExpandEnv(arg)
		}
		return RunWrapped(parts...)
	}

	return nil
}

type Procs struct {
}
