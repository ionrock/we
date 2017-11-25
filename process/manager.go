package process

import (
	"fmt"
	"os/exec"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/ionrock/we/process/forego"
)

type Manager struct {
	// processes is a map with our processes
	Procs map[string]*exec.Cmd

	// Output provides a prefix formatter for logging
	Output *forego.OutletFactory

	// wg for keeping track of our process go routines
	wg sync.WaitGroup

	teardown, teardownNow forego.Barrier
}

func NewManager(of *forego.OutletFactory) *Manager {
	return &Manager{
		Procs:  make(map[string]*exec.Cmd),
		Output: of,
	}
}

func (m *Manager) Processes() map[string]*exec.Cmd {
	return m.Procs
}

func (m *Manager) Start(name, command, dir string, env []string, of *forego.OutletFactory) error {
	if of == nil {
		of = m.Output
	}

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
	idx := len(m.Procs)
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
			m.teardown.Fall()

		case <-m.teardown.Barrier():
			of.SystemOutput(fmt.Sprintf("Killing %s", name))
			ps.Process.Kill()
		}
	}()

	m.Procs[name] = ps

	return nil
}

func (m *Manager) Stop(name string) error {
	svc, ok := m.Procs[name]
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
