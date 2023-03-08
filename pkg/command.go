package pkg

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/hashicorp/go-hclog"
)

// ConsulCommand interacts with a Consul binary on the system
type ConsulCommand struct {
	// ConsulBinary is the path on the system to the Consul binary used to invoke registration and connect commands.
	ConsulBinary string
	// Folder is the temporary folder to use in rendering out HCL files
	Folder string
	// Logger is the logger used for logging messages
	Logger hclog.Logger

	processes []*exec.Cmd
	mutex     sync.Mutex
}

func newCommand(binary string, logger hclog.Logger) (*ConsulCommand, error) {
	consul, err := findConsul(binary)
	if err != nil {
		return nil, err
	}

	folder, err := os.MkdirTemp("", "consul-services-*")
	if err != nil {
		return nil, err
	}

	cmd := &ConsulCommand{
		ConsulBinary: consul,
		Folder:       folder,
		Logger:       logger,
	}

	runtime.SetFinalizer(cmd, func(c *ConsulCommand) {
		cmd.Cleanup()
	})

	return cmd, nil
}

// Cleanup cleans up system resources after we're done
func (c *ConsulCommand) Cleanup() {
	c.mutex.Lock()
	for _, cmd := range c.processes {
		cmd.Cancel()
	}
	c.mutex.Unlock()
	os.RemoveAll(c.Folder)
}

func (c *ConsulCommand) runConsulBinary(ctx context.Context, logFn func(log string), args []string) error {
	output, err := os.CreateTemp(c.Folder, "process-*.log")
	if err != nil {
		return err
	}
	defer output.Close()

	if logFn != nil {
		logFn(output.Name())
	}

	var errBuffer bytes.Buffer
	writer := io.MultiWriter(&errBuffer, output)

	cmd := exec.CommandContext(ctx, c.ConsulBinary, args...)
	cmd.Stderr = writer
	cmd.Stdout = output

	c.mutex.Lock()
	c.processes = append(c.processes, cmd)
	c.mutex.Unlock()

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return errors.New(errBuffer.String())
		}
		return err
	}

	return nil
}

func findConsul(binary string) (string, error) {
	paths := []string{binary, defaultBinaryPath}
	path, err := exec.LookPath(binaryName)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err == nil {
		paths = append(paths, path)
	}

	for _, path := range paths {
		if path == "" {
			continue
		}

		found, normalized, err := checkConsulExecutable(path)
		if err != nil {
			return "", err
		}
		if found {
			return normalized, nil
		}
	}
	return "", errors.New("consul binary not found")
}

func checkConsulExecutable(path string) (bool, string, error) {
	info, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return false, "", err
	}

	if info == nil {
		return false, "", nil
	}

	normalized, err := filepath.Abs(path)
	if err != nil {
		return false, "", err
	}

	return err == nil && isExecutable(info), normalized, nil
}

func isExecutable(info fs.FileInfo) bool {
	return !info.IsDir() && (info.Mode()&0111 != 0)
}
