package pkg

import (
	"embed"
	"io/fs"
	"path"
	"text/template"
)

const (
	templatesDir = "templates"
)

const (
	protocolHTTP = "http"
	protocolTCP  = "tcp"

	externalServiceTemplate = "external-service.json"
	serviceTemplate         = "service.hcl"
	serviceDefaultsTemplate = "service-defaults.hcl"
	serviceProxyTemplate    = "service-proxy.hcl"
)

type templateArgs struct {
	*tracker

	// the id of the service
	ID string
	// the name of the service
	Name string
	// the port to register for the proxy
	ProxyPort int
	// the port that the service is served on
	ServicePort int
	// the protocol to use
	Protocol string
	// external upstreams to add
	ExternalUpstreams []*ConsulExternalService
}

var (
	//go:embed templates/*
	files     embed.FS
	templates map[string]*template.Template
)

func init() {
	templates = make(map[string]*template.Template)
	tmpls, err := fs.ReadDir(files, templatesDir)
	if err != nil {
		panic(err)
	}

	for _, tmpl := range tmpls {
		parsed := template.Must(template.ParseFS(files, path.Join(templatesDir, tmpl.Name())))
		templates[tmpl.Name()] = parsed
	}
}

func getTemplate(name string) *template.Template {
	return templates[name]
}
