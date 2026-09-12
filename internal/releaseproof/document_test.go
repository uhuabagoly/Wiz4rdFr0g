package releaseproof

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDocumentSignatureCoversNestedCommands(t *testing.T) {
	key := []byte(strings.Repeat("k",32))
	doc := map[string]any{"install_exit_code":0, "events":[]string{"real command output"}}
	b, _ := json.Marshal(doc)
	sig, err := DocumentSignature(b,key); if err != nil { t.Fatal(err) }
	doc["document_signature"] = sig
	b, _ = json.MarshalIndent(doc,"","  ")
	if err := VerifyDocument(b,key); err != nil { t.Fatal(err) }
	for _, field := range []string{"install_exit_code","events","environment"} {
		var changed map[string]any
		json.Unmarshal(b,&changed)
		changed[field] = "tampered"
		bad,_ := json.Marshal(changed)
		if VerifyDocument(bad,key) == nil { t.Fatalf("tampered %s accepted",field) }
	}
	if VerifyDocument(b,[]byte(strings.Repeat("z",32))) == nil { t.Fatal("wrong key accepted") }
}

func TestSignedBooleanOnlyPassIsNotPhysicalEvidence(t *testing.T) {
	key := []byte(strings.Repeat("k",32))
	doc := map[string]any{"final_status":"FULL_PASS", "install_verified":true,"uninstall_verified":true}
	b,_ := json.Marshal(doc); doc["document_signature"],_ = DocumentSignature(b,key)
	b,_ = json.Marshal(doc)
	if ValidatePhysicalDocument(b,key) == nil { t.Fatal("signed booleans must not count as physical proof") }
}
