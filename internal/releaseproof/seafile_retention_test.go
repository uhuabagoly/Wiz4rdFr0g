package releaseproof

import (
	"strings"
	"testing"
)

func TestSeafileRetentionRequiresBoundPreservedWitness(t *testing.T) {
	p := SeafileRetentionProof{Key: `HKEY_CURRENT_USER\SOFTWARE\Seafile`, View: "Registry64", Name: "Wiz4rdFr0g-retention-" + strings.Repeat("a", 32), Value: strings.Repeat("b", 64), Preserved: true}
	if !p.valid() {
		t.Fatal("valid witness rejected")
	}
	for _, change := range []func(*SeafileRetentionProof){func(p *SeafileRetentionProof) { p.Key += `\other` }, func(p *SeafileRetentionProof) { p.View = "Registry32" }, func(p *SeafileRetentionProof) { p.Name = "PreconfigureKeepConfigWhenUninstall" }, func(p *SeafileRetentionProof) { p.Value = "" }, func(p *SeafileRetentionProof) { p.Preserved = false }} {
		bad := p
		change(&bad)
		if bad.valid() {
			t.Fatal("invalid witness accepted")
		}
	}
	var missing *SeafileRetentionProof
	if missing.valid() {
		t.Fatal("missing witness accepted")
	}
}
