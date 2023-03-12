local zap = require("zap")
return function(config, endpoints)
  for _, endpoint in ipairs(endpoints) do
    zap.info("new endpoint", endpoint)
  end
end
