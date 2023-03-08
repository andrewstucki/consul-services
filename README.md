# Consul Services

This binary helps run testing services on the Consul Service mesh.

## Example

### API Gateway

In one window:

```bash
➜  consul-services --run --gateway example/gateways/api.hcl -e example/extra --http 3 -d 3
...
2023-03-08T00:53:50.545-0500 [INFO]  running gateway: admin=55388 port-one=55389 port-two=55390
```

And in another window:

```bash
➜  curl localhost:55389 -H "host: test.consul.local"
http-2-3
```

### Ingress Gateway

In one window:

```bash
➜  consul-services --run --gateway example/gateways/ingress.hcl
...
2023-03-08T00:57:57.422-0500 [INFO]  running gateway: admin=56901 port-one=56902 port-two=56903
```

And in another:

```bash
➜  curl localhost:56902
http-1-1
```

## Usage

```bash
➜  consul-services -h
Boots and registers a series of Consul service mesh services used in testing

Usage:
  consul-services [flags]

Flags:
      --consul string      Consul binary to use for registration, defaults to a binary found in the current folder and then the PATH.
  -d, --duplicates int     Number of duplicate services to register on the mesh. (default 1)
  -e, --extras string      Path to a folder containing extra configuration entries to write.
      --gateway string     Path to gateway definition to create, filed should be named 'api.hcl', 'ingress.hcl', etc. with a Port interpolation.
  -h, --help               help for consul-services
      --http int           Number of HTTP-based services to register on the mesh. (default 1)
  -r, --resources string   Folder of resources to apply, overrides tcp and http flags.
      --run                Additionally run Consul binary in agent mode.
      --tcp int            Number of TCP-based services to register on the mesh. (default 1)
```