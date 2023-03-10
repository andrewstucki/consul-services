# Features

- [x] Add a "report" command that dumps out the files used in the current run, with an optional `--script` output to turn everything into one giant shell script.
- [x] Allow the service runs to be daemonized similar to a `docker compose up -d`.
- [x] With the above, add a teardown command like `docker compose down`.
- [x] Add a log follower so we can look at logs of daemonized services.
- [x] Add an admin opener where we can immediately open the admin interface pages for any proxies.
- [x] Add the ability to spin up external services easily that can be routed through the mesh via terminating gateways (likely will need a paired trigger to act as a request proxy).
- [x] Add the ability to spin up resources in multiple consul dcs and have the dcs be connected via mesh gateways.
- [ ] Add the ability to specify the templates used for running target services.
- [ ] Add the ability to set timeout parameters on how services respond to requests.
- [ ] Add the ability to override dynamically allocated things like ports.

# Investigate

- [x] Figure out why failed spinups seem to orphan envoy processes (probably something to do with the way `exec.Command` is being used).