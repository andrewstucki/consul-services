primary_datacenter = "{{ .PrimaryDatacenter }}"
datacenter = "{{ .Datacenter }}"
addresses = {
  dns = "127.0.0.1"
  http = "127.0.0.1"
  grpc = "127.0.0.1"
  https = "127.0.0.1"
  grpc_tls = "127.0.0.1"
}
ports = {
  dns = {{ .GetNamedPort "dns" }}
  http = {{ .GetNamedPort "http" }}
  server = {{ .GetNamedPort "rpc" }}
  grpc = {{ .GetNamedPort "grpc" }}
  serf_lan = {{ .GetNamedPort "serf_lan" }}
  serf_wan = {{ .GetNamedPort "serf_wan" }}
  https = -1
  grpc_tls = -1
}