package pkg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/hashicorp/go-hclog"
)

// ConsulGateway is a gateway service to run on the Consul service mesh using the `consul connect envoy` command.
type ConsulGateway struct {
	// ConsulBinary is the path on the system to the Consul binary used to invoke registration and connect commands.
	ConsulBinary string
	// Name is the name of the gateway to run
	Name string
	// Kind is the kind of gateway to run
	Kind string
	// DefinitionFile is the path to the file used for registering the gateway
	DefinitionFile string
	// Folder is the temporary folder to use in rendering out HCL files
	Folder string
	// Logger is the logger used for logging messages
	Logger hclog.Logger

	// adminPort is the port allocated for envoy's admin interface
	adminPort int
	// gatewayPortOne is the first port allocated for the gateway
	gatewayPortOne int
	// gatewayPortTwo is the second port allocated for the gateway
	gatewayPortTwo int
	// gatewayPortThree is the third port allocated for the gateway
	gatewayPortThree int
}

func createGateway(binary, definition, folder string, logger hclog.Logger) (*ConsulGateway, error) {
	kind, name, err := getGatewayKindAndNameFromFile(definition)
	if err != nil {
		return nil, err
	}
	return &ConsulGateway{
		ConsulBinary:   binary,
		Kind:           kind,
		Name:           name,
		DefinitionFile: definition,
		Folder:         folder,
		Logger:         logger,
	}, nil
}

func getGatewayKindAndNameFromFile(file string) (string, string, error) {
	file = path.Base(file)

	switch {
	case strings.HasPrefix(file, "ingress"):
		return "ingress", strings.TrimSuffix(file, ".hcl"), nil
	case strings.HasPrefix(file, "terminating"):
		return "terminating", strings.TrimSuffix(file, ".hcl"), nil
	case strings.HasPrefix(file, "api"):
		return "api", strings.TrimSuffix(file, ".hcl"), nil
	case strings.HasPrefix(file, "mesh"):
		return "mesh", strings.TrimSuffix(file, ".hcl"), nil
	default:
		return "", "", errors.New("unsupported gateway type")
	}
}

// Run runs the Consul gateway
func (c *ConsulGateway) Run(ctx context.Context) error {
	if err := c.allocatePorts(); err != nil {
		return err
	}

	if err := c.renderGatewayDefinition(); err != nil {
		return err
	}

	if err := c.writeGatewayDefinition(ctx); err != nil {
		return err
	}

	return c.runEnvoy(ctx)
}

func (c *ConsulGateway) allocatePorts() error {
	adminPort, err := freePort()
	if err != nil {
		return err
	}
	gatewayPortOne, err := freePort()
	if err != nil {
		return err
	}
	gatewayPortTwo, err := freePort()
	if err != nil {
		return err
	}
	gatewayPortThree, err := freePort()
	if err != nil {
		return err
	}

	c.adminPort = adminPort
	c.gatewayPortOne = gatewayPortOne
	c.gatewayPortTwo = gatewayPortTwo
	c.gatewayPortThree = gatewayPortThree
	return nil
}

func (c *ConsulGateway) runEnvoy(ctx context.Context) error {
	c.Logger.Info("running gateway", "admin", c.adminPort, "port-one", c.gatewayPortOne, "port-two", c.gatewayPortTwo)

	return c.runConsulBinary(ctx, c.gatewayArgs())
}

func (c *ConsulGateway) writeGatewayDefinition(ctx context.Context) error {
	c.Logger.Info("writing gateway definition")

	return c.runConsulBinary(ctx, c.gatewayConfigArgs())
}

func (c *ConsulGateway) gatewayConfigArgs() []string {
	return []string{
		"config", "write",
		c.gatewayFile(),
	}
}

func (c *ConsulGateway) gatewayArgs() []string {
	args := []string{
		"connect", "envoy",
		"-gateway", c.Kind,
		"-register",
		"-service", c.Name,
		"-proxy-id", c.Name,
		"-admin-bind", fmt.Sprintf("127.0.0.1:%d", c.adminPort),
	}
	if c.Kind == "api" {
		args = append(args, "-address", fmt.Sprintf("127.0.0.1:%d", c.gatewayPortOne))
	}
	return args
}

func (c *ConsulGateway) runConsulBinary(ctx context.Context, args []string) error {
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

func (c *ConsulGateway) renderGatewayDefinition() error {
	return c.renderTemplate(c.DefinitionFile, c.gatewayFile())
}

func (c *ConsulGateway) gatewayFile() string {
	return path.Join(c.Folder, fmt.Sprintf("%s-gateway.hcl", c.Name))
}

type gatewayTemplateArgs struct {
	Name  string
	Port1 int
	Port2 int
	Port3 int
}

func (c *ConsulGateway) renderTemplate(template, name string) error {
	rendered, err := c.executeTemplate(template)
	if err != nil {
		return err
	}
	return os.WriteFile(name, rendered, 0600)
}

func (c *ConsulGateway) executeTemplate(name string) ([]byte, error) {
	var buffer bytes.Buffer

	file, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	template, err := template.New(name).Parse(string(file))
	if err != nil {
		return nil, err
	}

	if err := template.Execute(&buffer, &gatewayTemplateArgs{
		Name:  c.Name,
		Port1: c.gatewayPortOne,
		Port2: c.gatewayPortTwo,
		Port3: c.gatewayPortThree,
	}); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
