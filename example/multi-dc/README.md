# Multi-DC Example

This sets up a cluster with 3 HTTP service targets and 1 external HTTP service target each duplicated
across two federated datacenters: dc1 and dc2.

It sets up all services to resolve cross-DC queries through local mesh gateways and sets up a variety of
ingresses to test out.