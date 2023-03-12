return function(config)
  return {
    {
      hostname = "test-host",
      ipv4s = {"192.0.2.1"},
      ipv6s = {},
      record_ttl = 60,
      source_properties = nil,
      provider_properties = nil,
    },
  }
end
