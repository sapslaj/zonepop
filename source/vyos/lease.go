package vyos

import (
	"encoding/json"
	"strings"
)

type Lease struct {
	Pool            string   `json:"pool"`
	IP              string   `json:"ip"`
	Hostname        string   `json:"hostname"`
	HardwareAddress string   `json:"hardware_address"`
	IPv6s           []string `json:"ipv6s"`
}

func LeasesFromJSON(b []byte) ([]*Lease, error) {
	var leases []*Lease
	err := json.Unmarshal(b, &leases)
	if err != nil {
		return nil, err
	}
	return leases, nil
}

func (l *Lease) AssociatePotentialIPv6s(neighbors []*Neighbor) {
	for _, neighbor := range neighbors {
		if neighbor.To == "" {
			continue
		}
		if strings.HasPrefix(neighbor.To, "fe80") {
			continue
		}
		if !strings.EqualFold(neighbor.LLAddr, l.HardwareAddress) {
			continue
		}
		if neighbor.NUD == "REACHABLE" || neighbor.NUD == "STALE" {
			l.IPv6s = append(l.IPv6s, neighbor.To)
		}
	}
}
