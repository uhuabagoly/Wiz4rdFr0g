package releaseproof

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// DocumentSignature covers all fields, including command output and environment.
// Decode with UseNumber so integer exit codes retain their exact representation.
func DocumentSignature(data []byte, key []byte) (string, error) {
	if err := ValidateEvidenceKey(key); err != nil { return "", err }
	var doc map[string]any
	d := json.NewDecoder(bytes.NewReader(data)); d.UseNumber()
	if err := d.Decode(&doc); err != nil { return "", err }
	delete(doc, "document_signature")
	b, err := json.Marshal(doc); if err != nil { return "", err }
	h := hmac.New(sha256.New, key); h.Write(b)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func VerifyDocument(data []byte, key []byte) error {
	var doc struct { Signature string `json:"document_signature"` }
	if err := json.Unmarshal(data, &doc); err != nil { return err }
	want, err := DocumentSignature(data, key); if err != nil { return err }
	got, err := hex.DecodeString(doc.Signature); if err != nil { return err }
	expected, _ := hex.DecodeString(want)
	if !hmac.Equal(got, expected) { return fmt.Errorf("full evidence document signature invalid or missing") }
	return nil
}
