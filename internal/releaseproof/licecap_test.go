package releaseproof

import (
	"strings"
	"testing"
)

func TestLICEcapProofRequiresCompleteIndependentObservation(t *testing.T) {
	o := installedObservation{ID: "Cockos.LICEcap", State: "present", Command: []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", LICEcapIndependentScript}, Output: `{"state":"present","registry":"HKEY_LOCAL_MACHINE\\Software\\LICEcap","view":32,"registered_root":"C:\\Program Files (x86)\\LICEcap","files":[{"path":"C:\\Program Files (x86)\\LICEcap\\LICEcap.exe","bytes":485232,"pe":true},{"path":"C:\\Program Files (x86)\\LICEcap\\Uninstall.exe","bytes":51541,"pe":true}]}`}
	if !o.provesPresent() {
		t.Fatal("complete vendor proof rejected")
	}
	for _, mutate := range []func(*installedObservation){
		func(v *installedObservation) { v.Output = strings.ReplaceAll(v.Output, `"pe":true`, `"pe":false`) },
		func(v *installedObservation) { v.Output = strings.ReplaceAll(v.Output, `"view":32`, `"view":64`) },
		func(v *installedObservation) { v.Output = strings.ReplaceAll(v.Output, `Uninstall.exe`, `Other.exe`) },
		func(v *installedObservation) { v.Output = strings.ReplaceAll(v.Output, `Program Files (x86)`, `Users`) },
		func(v *installedObservation) {
			v.Command = []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "echo present"}
		},
		func(v *installedObservation) { v.ExitCode = 1 },
	} {
		bad := o
		mutate(&bad)
		if bad.provesPresent() {
			t.Fatal("incomplete or substituted proof accepted")
		}
	}
	o.State = "absent"
	o.Output = `{"state":"absent","registry":"HKEY_LOCAL_MACHINE\\Software\\LICEcap","view":32,"registered_root":null,"files":[]}`
	if !o.provesAbsent() {
		t.Fatal("complete absence rejected")
	}
	o.Output = strings.Replace(o.Output, `null`, `"C:\\Program Files (x86)\\LICEcap"`, 1)
	if o.provesAbsent() {
		t.Fatal("remaining registration accepted")
	}
	if (installedObservation{ID: "Other", State: "absent", ExitCode: 0}).provesAbsent() {
		t.Fatal("generic zero code accepted as absence")
	}
}
