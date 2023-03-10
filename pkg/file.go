package pkg

import (
	"bytes"
	"errors"
	"os"
	"text/template"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

type file struct {
	Kind string
	Name string
}

type dummyFileArgs struct {
	GetPort int
}

func newDummyArgs() *dummyFileArgs {
	return &dummyFileArgs{}
}

func (d *dummyFileArgs) GetCertificate(name string, sans ...string) *CertificateInfo {
	return &CertificateInfo{}
}

func (d *dummyFileArgs) GetNamedPort(name string) (int, error) {
	return 0, nil
}

func parseFile(path string) (*file, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// execute the template to get rid of interpolations
	var buffer bytes.Buffer
	tmpl, err := template.New(path).Parse(string(data))
	if err != nil {
		return nil, err
	}
	err = tmpl.Execute(&buffer, newDummyArgs())
	if err != nil {
		return nil, err
	}

	parser := hclparse.NewParser()
	f, diags := parser.ParseHCL(buffer.Bytes(), path)
	if diags.HasErrors() {
		return nil, diags
	}

	attrs, _ := f.Body.JustAttributes()

	parsed := &file{}

	for _, attr := range attrs {
		switch attr.Name {
		case "Name", "name":
			value, diags := attr.Expr.Value(nil)
			if diags.HasErrors() {
				return nil, diags
			}
			if value.Type() != cty.String {
				return nil, errors.New("invalid type for Name")
			}
			parsed.Name = value.AsString()
		case "Kind", "kind":
			value, diags := attr.Expr.Value(nil)
			if diags.HasErrors() {
				return nil, diags
			}
			if value.Type() != cty.String {
				return nil, errors.New("invalid type for Kind")
			}
			parsed.Kind = value.AsString()
		}
	}

	if parsed.Name == "" || parsed.Kind == "" {
		return nil, errors.New("unabled to parse Name and Kind")
	}

	return parsed, nil
}

func parseFileIntoEntry(server *server.Server, command *ConsulCommand, definition string, locality locality) (interface{}, error) {
	file, err := parseFile(definition)
	if err != nil {
		return nil, err
	}

	entry := &ConsulConfigEntry{
		ConsulCommand:  command,
		Kind:           file.Kind,
		Name:           file.Name,
		DefinitionFile: definition,
		Server:         server,
		tracker:        newTracker(),
		locality:       locality,
	}

	if _, ok := knownGateways[file.Kind]; ok {
		return &ConsulGateway{
			ConsulConfigEntry: entry,
			DefinitionFile:    definition,
		}, nil
	}

	return entry, nil
}
