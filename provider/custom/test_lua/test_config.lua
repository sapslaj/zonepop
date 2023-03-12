return function(config, endpoints)
  forward_endpoints = {}
  for i, endpoint in ipairs(endpoints) do
    print(string.format("[forward] %d: got %s", i, endpoint.hostname))
    if config.forward_lookup_filter(endpoint) then
      table.insert(forward_endpoints, endpoint)
      print(string.format("[forward] %d: inserted %s", i, endpoint.hostname))
    end
  end
  reverse_endpoints = {}
  for i, endpoint in ipairs(endpoints) do
    print(string.format("[reverse] %d: got %s", i, endpoint.hostname))
    if config.reverse_lookup_filter(endpoint) then
      table.insert(reverse_endpoints, endpoint)
      print(string.format("[reverse] %d: inserted %s", i, endpoint.hostname))
    end
  end
  if table.getn(forward_endpoints) ~= 1 then
    error(string.format("unexpected length for forward_endpoints (want 1, got %d)", table.getn(forward_endpoints)))
  end
  if table.getn(reverse_endpoints) ~= 1 then
    error(string.format("unexpected length for reverse_endpoints (want 1, got %d)", table.getn(reverse_endpoints)))
  end
  if forward_endpoints[1].hostname ~= "only-forward" then
    error(string.format("unexpected hostname '%s' was given", endpoint.hostname))
  end
  if reverse_endpoints[1].hostname ~= "only-reverse" then
    error(string.format("unexpected hostname '%s' was given", endpoint.hostname))
  end
end
