//go:build linux

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecorderForwardsVerifiedUpstreamBytes(t *testing.T) {
	payload := []byte{0, 1, 2, 3, 255, 0, 9}
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repo/objects/package" || r.URL.Query().Get("part") != "1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(payload)
	}))
	defer upstream.Close()
	previous := http.DefaultTransport
	http.DefaultTransport = upstream.Client().Transport
	defer func() { http.DefaultTransport = previous }()
	recorder, err := startPhysicalHTTPRecorder(upstream.URL + "/repo")
	if err != nil {
		t.Fatal(err)
	}
	defer recorder.close()
	response, err := http.Get(recorder.endpoint + "/objects/package?part=1")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || string(body) != string(payload) {
		t.Fatalf("body changed: %v %x", err, body)
	}
	recorder.close()
	records := recorder.snapshot()
	digest := sha256.Sum256(payload)
	if len(records) != 1 || records[0].Status != 200 || records[0].Bytes != int64(len(payload)) || records[0].SHA256 != hex.EncodeToString(digest[:]) || records[0].Error != "" {
		t.Fatalf("incorrect transfer proof: %+v", records)
	}
	if records[0].URL != upstream.URL+"/repo/objects/package?part=1" {
		t.Fatal(records[0].URL)
	}
}

func TestRecorderRequiresRealHTTPSOrigin(t *testing.T) {
	for _, origin := range []string{"file:///tmp", "http://example.com", "https://user:password@example.com"} {
		if recorder, err := startPhysicalHTTPRecorder(origin); err == nil {
			recorder.close()
			t.Fatal("unsafe origin accepted")
		}
	}
}
