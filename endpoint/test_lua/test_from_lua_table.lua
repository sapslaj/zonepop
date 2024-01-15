return function ()
  return {
    hostname = "test-host",
    ipv4s = { "192.0.2.1" },
    ipv6s = { "2001:db8::1" },
    record_ttl = 60,
    source_properties = {
      source_prop = "prop",
    },
    provider_properties = {
      provider_prop = "prop",
    },
  }
end
