local assert_equal = function(a, b)
  if not a == b then
    error(string.format("mismatch: '%s' != '%s'.", a, b))
  end
end

return function (endpoint)
  assert_equal(endpoint.hostname, "test-host")
  assert_equal(endpoint.ipv4s[1], "192.0.2.1")
  assert_equal(endpoint.ipv6s[1], "2001:db8::1")
  assert_equal(endpoint.record_ttl, 60)
  assert_equal(endpoint.source_properties.source_prop, "prop")
  assert_equal(endpoint.provider_properties.provider_prop, "prop")
end
