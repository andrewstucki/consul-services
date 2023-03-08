# Consul Services

This binary helps run testing services on the Consul Service mesh.

## Example

### API Gateway

In one window:

```bash
➜  consul-services -c example/consul-services.yaml
# or
➜  consul-services --run -r example --http 3 --tcp 3 -d 3
```

And in another window:

```bash
➜  GATEWAY_HTTP_PORT=$(consul-services get api -k api -f '.Ports[0]')
➜  curl localhost:$GATEWAY_HTTP_PORT -H "host: test.consul.local"
http-2-3
```

### Ingress Gateway

In one window:

```bash
➜  consul-services -c example/consul-services.yaml
# or
➜  consul-services --run -r example --http 3 --tcp 3 -d 3
```

And in another:

```bash
➜  INGRESS_HTTP_PORT=$(consul-services get ingress -k ingress -f '.Ports[0]')
➜  curl localhost:$INGRESS_HTTP_PORT
http-1-2
```

## Usage

```bash
➜  consul-services -h
Boots and registers a series of Consul service mesh services used in testing

Usage:
  consul-services [flags]
  consul-services [command]

Available Commands:
  admin       Opens the envoy admin panel for a given service.
  completion  Generate the autocompletion script for the specified shell
  get         Gets a particular service
  help        Help about any command
  list        Lists the services currently running.

Flags:
  -c, --config string      Path to configuration file. (default ".consul-services.yaml")
      --consul string      Consul binary to use for registration, defaults to a binary found in the current folder and then the PATH.
  -d, --duplicates int     Number of duplicate services to register on the mesh. (default 1)
  -h, --help               help for consul-services
      --http int           Number of HTTP-based services to register on the mesh. (default 1)
  -r, --resources string   Path to a folder containing extra configuration entries to write.
      --run                Additionally run Consul binary in agent mode.
  -s, --socket string      Path to unix socket for control server. (default "$HOME/.consul-services.sock")
      --tcp int            Number of TCP-based services to register on the mesh. (default 1)

Use "consul-services [command] --help" for more information about a command.
```