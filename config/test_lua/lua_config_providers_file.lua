return {
  providers = {
    file = {
      "file",
      config = {
        files = {
          {
            filename = "/etc/namedb/forward.zonepop.local.zone",
            permissions = "0644",
            zone = "zonepop.local.",
            generate = function (endpoints, ptr_records)
              local result = ""
              result = result .. "$ORIGIN zonepop.local.\n"
              result = result .. "$TTL 69\n"
              for _, endpoint in pairs(endpoints) do
                for _, ipv4 in pairs(endpoint.ipv4s or {}) do
                  result = result .. endpoint.hostname .. " A " .. ipv4 .. "\n"
                end
                for _, ipv6 in pairs(endpoint.ipv6s or {}) do
                  result = result .. endpoint.hostname .. " AAAA " .. ipv6 .. "\n"
                end
              end
              return result
            end,
          },
          {
            filename = "/etc/namedb/reverse.zonepop.local.zone",
            permissions = "0644",
            zone = "168.192.in-addr-arpa.",
            record_suffix = ".zonepop.local",
            template = [[
$ORIGIN {{ .FileConfig.Zone }}
$TTL 60
{{ range $ptr := .PTRRecords -}}
{{ $ptr.DomainName }} PTR {{ $ptr.Hostname }}
{{ end -}}
]],
          },
        },
      },
    },
  },
}
