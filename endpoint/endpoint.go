package endpoint

// Endpoint is a host configuration emitted from a source that contain all of
// the information for a provider to manage DNS records.
type Endpoint struct {
	// The hostname for the endpoint
	Hostname string `json:"hostname"`
	// List of IPv4 addresses for A record creation
	IPv4s []string `json:"ipv4s,omitempty"`
	// List of IPv6 addresses for AAAA record creation
	IPv6s []string `json:"ipv6s,omitempty"`
	// Preferred TTL for resulting records
	RecordTTL int64 `json:"ttl,omitempty"`
	// Additional key, value pairs from the source
	SourceProperties map[string]any `json:"source_properties,omitempty"`
	// Additional key, value pairs for the provider
	ProviderProperties map[string]any `json:"provider_properties,omitempty"`
}
