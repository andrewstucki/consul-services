{{ $certificate := .GetCertificate "*.consul.local" }}

Kind = "inline-certificate"
Name = "wildcard"
PrivateKey = <<EOF
{{ $certificate.PrivateKey }}
EOF
Certificate = <<EOF
{{ $certificate.Certificate }}
EOF
