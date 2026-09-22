package main

import "strings"

// Winget pads columns to the longest value plus one space. Splitting on
// repeated whitespace drops the longest (often only) result in each table.
func wingetTableRows(output string) [][]string {
	var previous string
	var starts []int
	var rows [][]string
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		// Progress redraws use a bare carriage return before the header.
		// Only the final visible line determines table column positions.
		if at := strings.LastIndex(raw, "\r"); at >= 0 {
			raw = raw[at+1:]
		}
		line := strings.TrimSpace(raw)
		if len(line) >= 3 && strings.Trim(line, "-") == "" {
			starts = nil
			inWord := false
			for i, r := range []rune(previous) {
				if r != ' ' && r != '\t' {
					if !inWord {
						starts = append(starts, i)
					}
					inWord = true
				} else {
					inWord = false
				}
			}
			continue
		}
		if len(starts) < 3 {
			previous = raw
			continue
		}
		if line == "" {
			continue
		}
		runes := []rune(raw)
		if len(runes) <= starts[2] {
			continue
		}
		var columns []string
		for i, start := range starts {
			end := len(runes)
			if i+1 < len(starts) && starts[i+1] < end {
				end = starts[i+1]
			}
			if start >= len(runes) {
				columns = append(columns, "")
				continue
			}
			columns = append(columns, strings.TrimSpace(string(runes[start:end])))
		}
		if columns[0] != "" && columns[1] != "" {
			rows = append(rows, columns)
		}
	}
	return rows
}
