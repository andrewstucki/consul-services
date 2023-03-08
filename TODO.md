# Features

- [ ] Add a "report" command that dumps out the files used in the current run, with an optional `--script` output to turn everything into one giant shell script.
- [ ] Allow the service runs to be daemonized similar to a `docker compose up -d`.
- [ ] With the above, add a teardown command like `docker compose down`.
- [ ] Add a log follower so we can look at logs of daemonized services.
- [ ] Add an admin opener where we can immediately open the admin interface pages for any proxies.

# Investigate

- [ ] Figure out why failed spinups seem to orphan envoy processes (probably something to do with the way `exec.Command` is being used).