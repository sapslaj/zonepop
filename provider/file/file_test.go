package file

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestFileProvider(t *testing.T) {
	t.Parallel()

	L := lua.NewState()
	defer L.Close()

	err := L.DoString(`
		return function (endpoints, ptr_records)
			local result = ""
			for _, endpoint in pairs(endpoints) do
				for _, ipv4 in pairs(endpoint.ipv4s or {}) do
					result = result .. endpoint.hostname .. " A " .. ipv4 .. "\n"
				end
				for _, ipv6 in pairs(endpoint.ipv6s or {}) do
					result = result .. endpoint.hostname .. " AAAA " .. ipv6 .. "\n"
				end
			end
			for _, ptr_record in pairs(ptr_records) do
				result = result .. ptr_record.domain_name .. " PTR " .. ptr_record.full_hostname .. "\n"
			end
			return result
		end
	`)
	require.NoError(t, err)

	generateFunc := L.Get(-1).(*lua.LFunction)

	template := `{{ range $endpoint := .Endpoints -}}
{{ range $ipv4 := $endpoint.IPv4s -}}
{{ $endpoint.Hostname }} A {{ $ipv4 }}
{{ end -}}
{{ range $ipv6 := $endpoint.IPv6s -}}
{{ $endpoint.Hostname }} AAAA {{ $ipv6 }}
{{ end -}}
{{ end -}}
{{ range $ptr := .PTRRecords -}}
{{ $ptr.DomainName }} PTR {{ $ptr.FullHostname }}
{{ end -}}`

	expect := `test-host A 192.0.2.1
test-host AAAA 2001:db8::1
1.2.0.192.in-addr.arpa. PTR test-host
1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa. PTR test-host
`

	tmpdir := t.TempDir()

	p, err := NewFileProvider(
		L,
		FileProviderConfig{
			Files: []FileProviderConfigFile{
				{
					Filename: path.Join(tmpdir, "generate"),
					Generate: generateFunc,
				},
				{
					Filename: path.Join(tmpdir, "template"),
					Template: template,
				},
			},
		},
		configtypes.DefaultEndpointFilterFunc,
		configtypes.DefaultEndpointFilterFunc,
	)
	require.NoError(t, err)

	err = p.UpdateEndpoints(context.Background(), []*endpoint.Endpoint{
		{
			Hostname:  "test-host",
			IPv4s:     []string{"192.0.2.1"},
			IPv6s:     []string{"2001:db8::1"},
			RecordTTL: 60,
		},
	})
	require.NoError(t, err)

	generatedData, err := os.ReadFile(path.Join(tmpdir, "generate"))
	require.NoError(t, err)
	assert.Equal(t, expect, string(generatedData))

	templatedData, err := os.ReadFile(path.Join(tmpdir, "template"))
	require.NoError(t, err)
	assert.Equal(t, expect, string(templatedData))
}
