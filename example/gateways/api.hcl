Kind = "api-gateway"
Name = "{{ .Name }}"
Listeners = [
  {
    Name     = "listener-one"
    Port     = {{ .Port1 }}
    Protocol = "http"
    Hostname = "*.consul.local"
  },
  {
    Name     = "listener-two"
    Port     = {{ .Port2 }}
    Protocol = "http"
    Hostname = "*.consul.local"
    TLS = {
      Certificates = [
        {
          Kind = "inline-certificate"
          Name = "wildcard"
        },
        {
          Kind = "inline-certificate"
          Name = "example"
        }
      ]
    }
  }
]