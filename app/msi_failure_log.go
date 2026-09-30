package main

import (
	"encoding/binary"
	"strings"
	"unicode/utf16"
)

// Windows Installer may emit UTF-16LE. Prefer the first failed action's
// immediate context over a long property dump at the end of its verbose log.
func msiFailureContext(data []byte) string {
	text := string(data)
	if len(data) >= 2 && data[0] == 0xff && data[1] == 0xfe {
		words := make([]uint16, 0, (len(data)-2)/2)
		for i := 2; i+1 < len(data); i += 2 {
			words = append(words, binary.LittleEndian.Uint16(data[i:i+2]))
		}
		text = string(utf16.Decode(words))
	}
	if end := strings.Index(strings.ToLower(text), "return value 3"); end >= 0 {
		end += len("return value 3")
		start := end - 4000
		if start < 0 {
			start = 0
		}
		return strings.TrimSpace(text[start:end])
	}
	// Avoid copying the MSI property table (which can contain user data).
	var errors []string
	for _, line := range strings.Split(text, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error ") || strings.Contains(lower, "failed") {
			if len(line) > 500 {
				line = line[:500]
			}
			errors = append(errors, strings.TrimSpace(line))
			if len(errors) == 8 {
				break
			}
		}
	}
	return strings.Join(errors, "\n")
}
