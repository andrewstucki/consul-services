package daemonize

import (
	"io"
	"os"
	"sync/atomic"

	"github.com/docker/docker/pkg/reexec"
)

var (
	longFlag  string
	shortFlag string
)

// SetDaemonizationFlag sets the flag to strip out when
// daemonization starts.
func SetDaemonizationFlag(long, short string) {
	longFlag = long
	shortFlag = short
}

func filterArgs(args []string) []string {
	filtered := []string{}
	for _, arg := range args {
		isLong := longFlag != "" && arg == longFlag
		isShort := shortFlag != "" && arg == shortFlag
		if !isLong && !isShort {
			filtered = append(filtered, arg)
		}
	}
	return filtered
}

func filterEnv(env []string) []string {
	filtered := []string{}
	for _, value := range env {
		if value != "_DAEMONIZE=1" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

// Daemonize sets up the double forking behavior needed
// to daemonize a process.
func Daemonize(args ...string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	cmd := reexec.Command(filterArgs(args)...)
	// clear out the flags to make the child terminate
	cmd.SysProcAttr = nil
	cmd.Env = append(os.Environ(), "_DAEMONIZE=1")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

// Handle sets up the second fork.
func Handle(args ...string) (bool, error) {
	daemonize := os.Getenv("_DAEMONIZE")
	if daemonize == "" {
		return false, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return false, err
	}

	cmd := reexec.Command(filterArgs(args)...)
	// clear out the flags that make the child terminate
	cmd.SysProcAttr = nil
	cmd.Env = filterEnv(os.Environ())
	cmd.Dir = dir
	stdout := newSingleLineWriter(os.Stderr)
	stderr := newSingleLineWriter(os.Stdout)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return true, err
	}

	select {
	case <-stdout.finished:
	case <-stderr.finished:
	}

	return true, nil
}

type singleLineWriter struct {
	writer   io.Writer
	finished chan struct{}
	written  int32
}

func newSingleLineWriter(w io.Writer) *singleLineWriter {
	return &singleLineWriter{
		writer:   w,
		finished: make(chan struct{}),
	}
}

func (s *singleLineWriter) Write(p []byte) (n int, err error) {
	if atomic.AddInt32(&s.written, 1) == 1 {
		n, err = s.writer.Write(p)
		close(s.finished)
	}
	return
}
