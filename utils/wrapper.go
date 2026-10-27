package utils

import (
	"bufio"
	"os"
	"os/exec"

	shlex "github.com/flynn/go-shlex"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

func splitCmd(cmd string) []string {
	// Split before expanding so variable values cannot inject extra arguments.
	parts, err := shlex.Split(cmd)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to split command")
	}
	for i := range parts {
		parts[i] = os.ExpandEnv(parts[i])
	}
	return parts
}

func RunLogged(parts ...string) error {
	log.Debug().Msgf("Running command: %v", parts)
	cmd := exec.Command(parts[0], parts[1:]...)

	o, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating stdout pipe")
	}

	e, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating stderr pipe")
	}

	stdout := bufio.NewScanner(o)
	stderr := bufio.NewScanner(e)
	go func() {
		for stdout.Scan() {
			log.Info().Msg(stdout.Text())
		}
	}()

	go func() {
		for stderr.Scan() {
			log.Info().Msg(stderr.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Error starting cmd")
	}

	return cmd.Wait()
}

func RunWrapped(parts ...string) error {
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func NewCommand(script string) *exec.Cmd {
	parts := splitCmd(script)
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
