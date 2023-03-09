{{ $certificate := .GetCertificate "example.consul.local" }}

Kind = "inline-certificate"
Name = "example"
PrivateKey = <<EOF
{{ $certificate.PrivateKey }}
EOF
Certificate = <<EOF
{{ $certificate.Certificate }}
EOF
