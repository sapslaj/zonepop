package vyos

import (
	"fmt"
	"strings"
)

type tabulateParseColumn struct {
	header string
	length int
	start  int
	end    int
}

func TabulateParse(b []byte) ([]map[string]string, error) {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid input: expected at least two lines")
	}
	columns := []tabulateParseColumn{}
	currentColumnLength := 0
	for i := range len(lines[1]) {
		if lines[1][i] == '-' {
			currentColumnLength += 1
		} else if currentColumnLength > 0 {
			columns = append(columns, tabulateParseColumn{
				length: currentColumnLength,
				start:  i - currentColumnLength,
				end:    i,
			})
			currentColumnLength = 0
		}
	}
	columns = append(columns, tabulateParseColumn{
		length: currentColumnLength,
		start:  len(lines[1]) - currentColumnLength,
		end:    len(lines[1]),
	})

	for i, column := range columns {
		columns[i].header = strings.TrimSpace(lines[0][column.start:min(len(lines[0]), column.end)])
	}

	rows := []map[string]string{}
	for _, line := range lines[2:] {
		row := map[string]string{}
		for _, column := range columns {
			row[column.header] = strings.TrimSpace(line[column.start:min(len(line), column.end)])
		}
		rows = append(rows, row)
	}
	return rows, nil
}
