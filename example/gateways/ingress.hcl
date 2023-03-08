Kind = "ingress-gateway"
Name = "{{ .Name }}"
Listeners = [
  {
    Port = {{ .Port1 }}
    Protocol = "http"
    Services = [{
      Name = "http-1"
      Hosts = ["*"]
    }]
  }
]