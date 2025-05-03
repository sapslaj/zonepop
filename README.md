# ZonePop

[![CI](https://github.com/sapslaj/zonepop/actions/workflows/ci.yaml/badge.svg)](https://github.com/sapslaj/zonepop/actions/workflows/ci.yaml)
[![Coverage Status](https://coveralls.io/repos/github/sapslaj/zonepop/badge.svg?branch=main)](https://coveralls.io/github/sapslaj/zonepop?branch=main)
[![GitHub release](https://img.shields.io/github/release/sapslaj/zonepop.svg)](https://github.com/sapslaj/zonepop/releases)
[![go-doc](https://godoc.org/github.com/sapslaj/zonepop?status.svg)](https://godoc.org/github.com/sapslaj/zonepop)
[![Go Report Card](https://goreportcard.com/badge/github.com/sapslaj/zonepop)](https://goreportcard.com/report/github.com/sapslaj/zonepop)

Dynamic DNS for the lazy sysadmin.

## What It Does

Inspired by [ExternalDNS](https://github.com/kubernetes-sigs/external-dns), ZonePop is a DNS syncing service that takes DHCP leases and IPv6 neighbors from any number of sources and syncs those to a DNS provider. No RFC 2136, client configuration, or complex split-horizon DNS resolver setup required (unless you feel like it, of course).

ZonePop is still in the early stages of development (and might be for a long time since it's just a single person maintaining it), so the number of sources and providers is limited for now.

### Sources

- `custom` - Arbitrary Lua function
- `vyos_ssh` - VyOS DHCP leases fetched via SSH

### Providers

- `aws_route53` - Updates records in a AWS Route53 hosted zone
- `custom` - Arbitrary Lua function
- `hosts_file` - Generates an `/etc/hosts` style file, optionally uploading to remote server via SSH
- `http` - Exposes a JSON list representation accessible via the `/endpoints` HTTP endpoint.
- `prometheus_metrics` - Exports info metrics for each endpoint in Prometheus format, accessible via the `/metrics` HTTP endpoint.

## Configuration

The configuration file for ZonePop is a [Lua](https://www.lua.org/) script. Why Lua instead of YAML or similar? Lua allows for much more flexibility over how DNS records are created and managed.

A simple config file looks something like this:

```lua
return {
  sources = {
    vyos = {
      "vyos_ssh",
      config = {
        host = os.getenv("VYOS_HOST"),
        username = os.getenv("VYOS_USERNAME"),
        password = os.getenv("VYOS_PASSWORD"),
      },
    },
  },
  providers = {
    route53 = {
      "aws_route53",
      config = {
        record_suffix = ".example.com",
        forward_zone_id = "Z2FDTNDATAQYW2",
      },
    },
  },
}
```

The main config file should return a Table with the `sources` and `providers` keys. The keys for those sub-tables are simply logical names. The first value in each of those tables is the kind. For example, the Route53 provider uses the `aws_route53` kind. The next key, `config` is the configuration for that source or provider. This will vary based on the source and provider (docs TBD).
