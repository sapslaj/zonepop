package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
)

func TestHTTPProvider(t *testing.T) {
	t.Parallel()

	p, err := NewHTTPProvider(
		HTTPProviderConfig{},
		configtypes.DefaultEndpointFilterFunc,
	)
	require.NoError(t, err)

	server := httptest.NewServer(http.DefaultServeMux)
	defer server.Close()

	res, err := http.Get(server.URL + "/endpoints/forward")
	require.NoError(t, err)
	bodyData, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	res.Body.Close()
	var emptyResult []*endpoint.Endpoint
	err = json.Unmarshal(bodyData, &emptyResult)
	require.NoError(t, err)

	assert.Len(t, emptyResult, 0)

	p.UpdateEndpoints(context.Background(), []*endpoint.Endpoint{
		{
			Hostname:  "test-host",
			IPv4s:     []string{"192.0.2.1"},
			IPv6s:     []string{"2001:db8::1"},
			RecordTTL: 60,
		},
	})

	res, err = http.Get(server.URL + "/endpoints/forward")
	require.NoError(t, err)
	bodyData, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	res.Body.Close()
	var populatedResult []*endpoint.Endpoint
	err = json.Unmarshal(bodyData, &populatedResult)
	require.NoError(t, err)

	assert.Len(t, populatedResult, 1)
	assert.Equal(t, []*endpoint.Endpoint{
		{
			Hostname:  "test-host",
			IPv4s:     []string{"192.0.2.1"},
			IPv6s:     []string{"2001:db8::1"},
			RecordTTL: 60,
		},
	}, populatedResult)
}
