package main

import (
	"encoding/binary"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestMSIFailureContextDecodesAndBoundsVendorFailure(t *testing.T) {
	text := strings.Repeat("setup detail\n", 600) + "custom action failed\nAction ended: RemoveService. Return value 3\nProperty(S): PRIVATE=not-needed"
	data := []byte{0xff, 0xfe}
	for _, v := range utf16.Encode([]rune(text)) {
		data = binary.LittleEndian.AppendUint16(data, v)
	}
	got := msiFailureContext(data)
	if len(got) > 4000 || !strings.Contains(got, "custom action failed") || strings.Contains(got, "PRIVATE") {
		t.Fatalf("incorrect failure context: %q", got)
	}
	if got := msiFailureContext([]byte("Property(S): PRIVATE=not-needed\nError 1722: action failed\n")); got != "Error 1722: action failed" {
		t.Fatalf("unexpected fallback: %q", got)
	}
}
