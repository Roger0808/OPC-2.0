package cloud

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBatchDownloadPOST(t *testing.T) {
	const body = `{"ids":["one","two"]}`
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if r.Method != "POST" || string(data) != body || r.Header.Get("Authorization") != "Bearer token" || r.Header.Get("X-Tenant-ID") != "42" {
			t.Error("batch download contract changed")
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Write([]byte("zip-bytes"))
	}))
	defer up.Close()
	resp, err := DownloadRequest(context.Background(), up.URL, "/api/v1/knowledge-bases/k/knowledge/batch-download", "token", "42", "", "POST", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if string(data) != "zip-bytes" {
		t.Fatal("batch response altered")
	}
	if _, err = DownloadRequest(context.Background(), up.URL, "/api/v1/auth/register", "token", "42", "", "POST", body); err == nil {
		t.Fatal("arbitrary POST allowed")
	}
}
