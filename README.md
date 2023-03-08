# Consul Services

This binary helps run testing services on the Consul Service mesh.

## Example

### API Gateway

In one window:

```bash
➜  consul-services -c example/consul-services.yaml
# or
➜  consul-services --run -r example --http 3 --tcp 3 -d 3
...
2023-03-08T10:40:08.341-0500 [INFO]  running gateway: admin=55829 ports=[55830, 55831]
...
```

And in another window:

```bash
➜  curl localhost:55830 -H "host: test.consul.local"
http-2-3
```

### Ingress Gateway

In one window:

```bash
➜  consul-services --run -r example --http 3 -d 3
...
2023-03-08T10:40:08.749-0500 [INFO]  running gateway: admin=55862 ports=[55863]
...
```

And in another:

```bash
➜  curl localhost:55863
http-1-1
```

## Usage

```bash
➜  consul-services -h
Boots and registers a series of Consul service mesh services used in testing

Usage:
  consul-services [flags]

Flags:
  -c, --config string      Path to configuration file. (default ".consul-services.yaml")
      --consul string      Consul binary to use for registration, defaults to a binary found in the current folder and then the PATH.
  -d, --duplicates int     Number of duplicate services to register on the mesh. (default 1)
  -h, --help               help for consul-services
      --http int           Number of HTTP-based services to register on the mesh. (default 1)
  -r, --resources string   Path to a folder containing extra configuration entries to write.
      --run                Additionally run Consul binary in agent mode.
      --tcp int            Number of TCP-based services to register on the mesh. (default 1)
```