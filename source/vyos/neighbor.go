package vyos

import (
	"strings"
)

type Neighbor struct {
	To     string
	Dev    string
	LLAddr string
	NUD    string
}

var ValidNUDs = []string{
	"PERMANENT",
	"NOARP",
	"REACHABLE",
	"STALE",
	"NONE",
	"INCOMPLETE",
	"DELAY",
	"PROBE",
	"FAILED",
}

func ParseNeighborLines(lines string) ([]*Neighbor, error) {
	neighbors := make([]*Neighbor, 0)
	for _, line := range strings.Split(lines, "\n") {
		neighbor, err := ParseNeighborLine(line)
		if err != nil {
			return neighbors, err
		}
		neighbors = append(neighbors, neighbor)
	}
	return neighbors, nil
}

func ParseNeighborLine(line string) (*Neighbor, error) {
	neighbor := &Neighbor{}
	parts := strings.Split(line, " ")
	for i := 0; i < len(parts); i++ {
		if i == 0 {
			neighbor.To = parts[i]
			continue
		}
		if parts[i] == "dev" {
			i++
			neighbor.Dev = parts[i]
			continue
		}
		if parts[i] == "lladdr" {
			i++
			neighbor.LLAddr = parts[i]
			continue
		}
		maybeNUD := strings.ToUpper(parts[i])
		for _, validNUD := range ValidNUDs {
			if validNUD == maybeNUD {
				neighbor.NUD = maybeNUD
				continue
			}
		}
	}
	return neighbor, nil
}
