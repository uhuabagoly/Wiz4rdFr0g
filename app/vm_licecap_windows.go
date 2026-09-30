//go:build windows

package main

import (
	"context"
	"encoding/json"
	"time"

	"wiz4rdfr0g.local/fullcatalog/internal/releaseproof"
)

// LICEcap has no ARP/WinGet inventory registration. Independently inspect its
// publisher-defined registry value and both PE files through .NET, without
// calling the production registry adapter or its path/parser helpers.
func vmIndependentLICEcapProbe() vmIndependentState {

	args := []string{"-NoProfile", "-NonInteractive", "-Command", releaseproof.LICEcapIndependentScript}
	r := vmIndependentState{ID: "Cockos.LICEcap", State: "unknown", Command: append([]string{"powershell.exe"}, args...)}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	code, out, err := runDirectProcess(ctx, "powershell.exe", args)
	r.ExitCode, r.Output = code, out
	if err != nil {
		r.Error = err.Error()
		return r
	}
	var observation struct {
		State string `json:"state"`
	}
	if code == 0 && json.Unmarshal([]byte(out), &observation) == nil && (observation.State == "present" || observation.State == "absent") {
		r.State = observation.State
	}
	return r
}
