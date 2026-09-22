//go:build linux

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type physicalTransfer struct {
	URL         string `json:"resolved_download_url"`
	FinalURL    string `json:"final_download_url"`
	Status      int    `json:"download_http_status"`
	Bytes       int64  `json:"downloaded_bytes"`
	SHA256      string `json:"sha256"`
	ContentType string `json:"content_type"`
	Error       string `json:"error,omitempty"`
	Started     string `json:"started_at"`
	Finished    string `json:"finished_at"`
}

// Only the disposable runner's configured Flatpak repository uses this recorder.
// Every response body is fetched from its original HTTPS origin, with normal TLS
// verification, then forwarded byte-for-byte. OSTree still verifies signatures.
// No fabricated package, local fixture or system-wide interception is involved.
type physicalHTTPRecorder struct {
	server    *http.Server
	endpoint  string
	mu        sync.Mutex
	transfers []physicalTransfer
}

func startPhysicalHTTPRecorder(origin string) (*physicalHTTPRecorder, error) {
	base, err := url.Parse(origin)
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil {
		return nil, fmt.Errorf("Flatpak recorder requires an HTTPS repository origin")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	recorder := &physicalHTTPRecorder{endpoint: "http://" + listener.Addr().String()}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableCompression = true
	client := &http.Client{Transport: transport, Timeout: 20 * time.Minute}
	recorder.server = &http.Server{ReadHeaderTimeout: 10 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != "GET" && request.Method != "HEAD" {
			http.Error(w, "unsupported method", 405)
			return
		}
		target := *base
		target.Path = strings.TrimRight(base.Path, "/") + request.URL.Path
		target.RawQuery = request.URL.RawQuery
		record := physicalTransfer{URL: target.String(), Started: time.Now().UTC().Format(time.RFC3339Nano)}
		defer func() {
			record.Finished = time.Now().UTC().Format(time.RFC3339Nano)
			recorder.mu.Lock()
			recorder.transfers = append(recorder.transfers, record)
			recorder.mu.Unlock()
		}()
		upstream, err := http.NewRequestWithContext(request.Context(), request.Method, target.String(), nil)
		if err != nil {
			record.Error = err.Error()
			http.Error(w, "request failed", 502)
			return
		}
		for _, key := range []string{"Range", "If-None-Match", "If-Modified-Since"} {
			if value := request.Header.Get(key); value != "" {
				upstream.Header.Set(key, value)
			}
		}
		response, err := client.Do(upstream)
		if err != nil {
			record.Error = err.Error()
			http.Error(w, "upstream failed", 502)
			return
		}
		defer response.Body.Close()
		record.Status = response.StatusCode
		record.FinalURL = response.Request.URL.String()
		record.ContentType = response.Header.Get("Content-Type")
		for _, key := range []string{"Content-Type", "Content-Length", "Content-Encoding", "ETag", "Last-Modified", "Content-Range"} {
			if value := response.Header.Get(key); value != "" {
				w.Header().Set(key, value)
			}
		}
		w.WriteHeader(response.StatusCode)
		hash := sha256.New()
		record.Bytes, err = io.Copy(io.MultiWriter(w, hash), response.Body)
		record.SHA256 = hex.EncodeToString(hash.Sum(nil))
		if err != nil {
			record.Error = err.Error()
		}
	})}
	go recorder.server.Serve(listener)
	return recorder, nil
}

func (r *physicalHTTPRecorder) close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r.server.Shutdown(ctx)
}

func (r *physicalHTTPRecorder) snapshot() []physicalTransfer {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]physicalTransfer(nil), r.transfers...)
}

// Flatpak extra-data is downloaded from vendor URLs outside the OSTree repo.
// Verify every source against the exact commit's published size and checksum.
func verifyFlatpakExtraData(ctx context.Context, metadata, work string) ([]physicalTransfer, error) {
	fields := map[string]string{}
	inExtra := false
	for _, line := range strings.Split(metadata, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			inExtra = line == "[Extra Data]"
			continue
		}
		if inExtra {
			if key, value, ok := strings.Cut(line, "="); ok {
				fields[key] = value
			}
		}
	}
	var records []physicalTransfer
	for key, source := range fields {
		if !strings.HasPrefix(key, "uri") {
			continue
		}
		suffix := strings.TrimPrefix(key, "uri")
		expected := fields["checksum"+suffix]
		size, err := strconv.ParseInt(fields["size"+suffix], 10, 64)
		if err != nil || size <= 0 || len(expected) != 64 {
			return records, fmt.Errorf("Flatpak extra-data has no exact size/checksum")
		}
		uri, err := url.Parse(source)
		if err != nil || (uri.Scheme != "https" && uri.Scheme != "http") || uri.Host == "" {
			return records, fmt.Errorf("invalid extra-data URL")
		}
		record := physicalTransfer{URL: source, Started: time.Now().UTC().Format(time.RFC3339Nano)}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return records, err
		}
		response, err := (&http.Client{Timeout: 20 * time.Minute}).Do(request)
		if err != nil {
			return records, err
		}
		record.Status = response.StatusCode
		record.FinalURL = response.Request.URL.String()
		record.ContentType = response.Header.Get("Content-Type")
		if response.StatusCode != 200 || strings.Contains(strings.ToLower(record.ContentType), "text/html") {
			response.Body.Close()
			return records, fmt.Errorf("extra-data HTTP/content type invalid")
		}
		file, err := os.Create(filepath.Join(work, fmt.Sprintf("extra-data-%d", len(records))))
		if err != nil {
			response.Body.Close()
			return records, err
		}
		hash := sha256.New()
		record.Bytes, err = io.Copy(io.MultiWriter(file, hash), response.Body)
		response.Body.Close()
		file.Close()
		record.SHA256 = hex.EncodeToString(hash.Sum(nil))
		record.Finished = time.Now().UTC().Format(time.RFC3339Nano)
		records = append(records, record)
		if err != nil {
			return records, err
		}
		if record.Bytes != size || record.SHA256 != expected {
			return records, fmt.Errorf("Flatpak extra-data size/checksum mismatch")
		}
	}
	return records, nil
}
