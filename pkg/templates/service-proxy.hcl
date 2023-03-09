Service {
  Kind = "connect-proxy"
  Name = "{{ .Name }}-proxy"
  ID   = "{{ .ID }}-proxy"
  Port = {{ .ProxyPort }}

  proxy = {
    destination_service_name  = "{{ .Name }}"
    destination_service_id    = "{{ .ID }}"
    local_service_address     = "127.0.0.1"
    local_service_port        = {{ .ServicePort }}
    {{ $service := . }}
    {{- range $upstream := .ExternalUpstreams }}
    upstreams {
      destination_name = "{{ $upstream }}"
      local_bind_port = {{ $service.GetNamedPort $upstream }}
    }
    {{- end }}
  }
}