# Consul Services

This binary helps run testing services on the Consul Service mesh.

## Example

Boot up the services in the background:

```bash
consul-services -c example/consul-services.yaml -d
# or
consul-services --run -r example --http 3 --external-http 1 -D 3 -d
```

Test the API gateway:

```bash
GATEWAY_HTTP_PORT=$(consul-services get api -k api -f '.NamedPorts.one')
curl localhost:$GATEWAY_HTTP_PORT -H "host: test.consul.local"
```

Test the ingress gateway:

```bash
INGRESS_HTTP_PORT=$(consul-services get ingress -k ingress -f '.Ports[0]')
curl localhost:$INGRESS_HTTP_PORT
```

Test connectivity through the terminating gateway:

```bash
consul-services check http-1-1 http-external-1
```

Open up the admin interface of the API gateway:

```bash
consul-services admin api -k api
```

Check the logs of the API gateway:

```bash
consul-services logs api -k api
```

List all services:

```bash
consul-services list -a
```

Stop the services

```bash
consul-services stop
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
  check       Checks for one-way connectivity between two services
  completion  Generate the autocompletion script for the specified shell
  get         Gets a particular service
  help        Help about any command
  list        Lists the services currently running.
  logs        Read logs from a deployed service.
  stop        Stops a daemonized run

Flags:
  -c, --config string       Path to configuration file. (default ".consul-services.yaml")
      --consul string       Consul binary to use for registration, defaults to a binary found in the current folder and then the PATH.
  -d, --daemon              Daemonize the process.
  -D, --duplicates int      Number of duplicate services to register on the mesh. (default 1)
      --external-http int   Number of HTTP-based external services to register on the mesh.
      --external-tcp int    Number of TCP-based external services to register on the mesh.
  -h, --help                help for consul-services
      --http int            Number of HTTP-based services to register on the mesh. (default 1)
  -o, --output string       Path to use for output rather than stdout.
  -r, --resources string    Path to a folder containing extra configuration entries to write.
      --run                 Additionally run Consul binary in agent mode.
  -s, --socket string       Path to unix socket for control server. (default "$HOME/.consul-services.sock")
      --tcp int             Number of TCP-based services to register on the mesh.

Use "consul-services [command] --help" for more information about a command.
```

## Catastrophes

If there are any bugs that you are hitting that are leaving processes in an orphaned state (this binary does a lot of child execs, and so does `consul connect` itself), then the easiest way to kill anything this may spin up and get your system back in a normal state (assuming you don't have any instances of `consul` or `envoy` running that you want to keep around) is (on a Mac):

```bash
killall consul; killall envoy; killall consul-services; rm ~/.consul-services.sock
```