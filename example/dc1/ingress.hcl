Kind = "ingress-gateway"
Name = "ingress"
Listeners = [
  {
    Port = {{ .GetPort }}
    Protocol = "http"
    Services = [{
      Name = "http-1"
      Hosts = ["*"]
    },{
      Name = "http-dc-2"
      Hosts = ["dc2.consul.internal"]
    }]
  }
]