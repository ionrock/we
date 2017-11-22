package process

import (
	"fmt"
	"os/exec"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/ionrock/we/process/forego"
)

type Manager struct {
	// Processes is a map with our processes
	Processes map[string]*exec.Cmd

	// Output provides a prefix formatter for logging
	Output *forego.OutletFactory

	// wg for keeping track of our process go routines
	wg sync.WaitGroup

	teardown, teardownNow forego.Barrier
}

func (m *Manager) Start(name, command, dir string, env []string, of *forego.OutletFactory) error {
	parts := SplitCommand(command)

	var ps *exec.Cmd
	if len(parts) == 1 {
		ps = exec.Command(parts[0])
	} else {
		ps = exec.Command(parts[0], parts[1:]...)
	}

	if dir != "" {
		ps.Dir = dir
	}

	if env != nil {
		ps.Env = env
	}

	stdout, err := ps.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := ps.StderrPipe()
	if err != nil {
		return err
	}

	// Start reading the output of the
	pipeWait := new(sync.WaitGroup)
	pipeWait.Add(2)
	idx := len(m.Processes)
	go of.LineReader(pipeWait, name, idx, stdout, false)
	go of.LineReader(pipeWait, name, idx, stderr, true)

	of.SystemOutput(fmt.Sprintf("starting %s on port TODO", name))

	finished := make(chan struct{}) // closed on process exit

	err = ps.Start()
	if err != nil {
		of.SystemOutput(fmt.Sprint("Failed to start ", name, ": ", err))
		return err
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(finished)
		pipeWait.Wait()
		ps.Wait()
	}()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		select {
		case <-finished:
			// TODO: this was to handle restarts in forego
			// if flagRestart {
			// 	m.startProcess(idx, procNum, proc, env, of)
			// } else {
			m.teardown.Fall()
			// }

		case <-m.teardown.Barrier():
			// Forego tearing down

			of.SystemOutput(fmt.Sprintf("Killing %s", name))
			ps.Process.Kill()
		}
	}()

	m.Processes[name] = ps

	return nil
}

func (m *Manager) Stop(name string) error {
	svc, ok := m.Processes[name]
	if !ok {
		// should probably still throw an error here...
		return nil
	}

	if svc.ProcessState != nil {
		return nil
	}

	err := svc.Process.Kill()
	if err != nil {
		log.Printf("error killing service: %s, %s\n", name, err)
		return err
	}

	return nil
}
