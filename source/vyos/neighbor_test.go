package vyos

import "testing"

func TestParseNeighborLine(t *testing.T) {
	tests := []struct {
		input  string
		to     string
		dev    string
		lladdr string
		nud    string
	}{
		{
			input:  "2001:470:e022:5:e4d9:f53b:a364:b510 dev eth0.5 lladdr a0:36:bc:8b:b7:ca STALE",
			to:     "2001:470:e022:5:e4d9:f53b:a364:b510",
			dev:    "eth0.5",
			lladdr: "a0:36:bc:8b:b7:ca",
			nud:    "STALE",
		},
		{
			input: "2001:470:e022:5:468d:bd94:fead:b84c dev eth0.5  FAILED",
			to:    "2001:470:e022:5:468d:bd94:fead:b84c",
			dev:   "eth0.5",
			nud:   "FAILED",
		},
	}

	for _, tc := range tests {
		n, err := ParseNeighborLine(tc.input)
		if err != nil {
			t.Fatalf("input: %q, err: %v", tc.input, err)
		}
		if n.To != tc.to {
			t.Fatalf("input: %q, n.To == %s ; expected `%s`", tc.input, n.To, tc.to)
		}
		if n.Dev != tc.dev {
			t.Fatalf("input: %q, n.Dev == %s ; expected `%s`", tc.input, n.Dev, tc.dev)
		}
		if n.LLAddr != tc.lladdr {
			t.Fatalf("input: %q, n.LLAddr == %s ; expected `%s`", tc.input, n.LLAddr, tc.lladdr)
		}
		if n.NUD != tc.nud {
			t.Fatalf("input: %q, n.NUD == %s ; expected `%s`", tc.input, n.NUD, tc.nud)
		}
	}
}
