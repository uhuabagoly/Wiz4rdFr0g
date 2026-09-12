package main

import "regexp"

func independentWingetState(id string, code int, output string, commandFailed bool) string {
	if uint32(code) == 0x8a150014 { return "absent" }
	if id != "" && !commandFailed && code == 0 && regexp.MustCompile(`(?m)\s`+regexp.QuoteMeta(id)+`\s`).MatchString(output) { return "present" }
	return "unknown"
}
