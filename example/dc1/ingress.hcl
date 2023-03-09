Kind = "ingress-gateway"
Name = "ingress"
Listeners = [
  {
    Port = {{ .GetPort }}
    Protocol = "http"
    Services = [{
      Name = "http-1"
      Hosts = ["*"]
    }]
  }
]