return {
  providers = {
    custom = {
      "custom",
      config = {
        update_endpoints = function(config, endpoints)
          filtered_endpoints = {}
          for i, endpoint in ipairs(endpoints) do
            if config.reverse_lookup_filter(endpoint) then
              table.insert(filtered_endpoints, endpoint)
            end
          end
          if table.getn(filtered_endpoints) ~= 1 then
            error(string.format("unexpected length for forward_endpoints (want 1, got %d)", table.getn(forward_endpoints)))
          end
          if filtered_endpoints[1].hostname ~= "host-1" then
            error(string.format("unexpected hostname '%s' was given", endpoint.hostname))
          end
        end,
        reverse_lookup_filter = function(endpoint)
          return endpoint.hostname == "host-1"
        end,
      }
    }
  }
}
