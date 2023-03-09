Kind = "api-gateway"
Name = "api"
Listeners = [
  {
    Name     = "listener-one"
    Port     = {{ .GetNamedPort "one" }}
    Protocol = "http"
    Hostname = "*.consul.local"
  },
  {
    Name     = "listener-two"
    Port     = {{ .GetNamedPort "two" }}
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