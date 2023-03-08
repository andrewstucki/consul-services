package pkg

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

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

	return &ConsulCommand{
		ConsulBinary: consul,
		Folder:       folder,
		Logger:       logger,
	}, nil
}

// Cleanup cleans up system resources after we're done
func (c *ConsulCommand) Cleanup() {
	os.RemoveAll(c.Folder)
}

func (c *ConsulCommand) runConsulBinary(ctx context.Context, args []string) error {
	var errBuffer bytes.Buffer

	cmd := exec.CommandContext(ctx, c.ConsulBinary, args...)
	cmd.Stderr = &errBuffer

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
