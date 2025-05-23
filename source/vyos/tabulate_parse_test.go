package vyos

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTabulateParse(t *testing.T) {
	t.Parallel()

	data := `
Name   Gender  Age
-----  ------  ---
Alice  F       24
Bob    M       19
Tay            21
`
	data = strings.TrimPrefix(data, "\n")

	expect := []map[string]string{
		{
			"Name": "Alice",
			"Gender": "F",
			"Age": "24",
		},
		{
			"Name": "Bob",
			"Gender": "M",
			"Age": "19",
		},
		{
			"Name": "Tay",
			"Gender": "",
			"Age": "21",
		},
	}

	got, err := TabulateParse([]byte(data))
	require.NoError(t, err)
	require.Equal(t, expect, got)
}
