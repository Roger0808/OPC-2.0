package cloud

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestOriginAndResourceValidation(t *testing.T) {
	for _, raw := range []string{"https://user:password@example.com", "https://example.com/api/v1", "https://example.com?x=1", "http://example.com", "file:///etc/passwd", "https://example.com#fragment", "https://example.com/../"} {
		if _, err := NormalizeOrigin(raw, true); err == nil {
			t.Fatalf("accepted invalid origin %s", raw)
		}
	}
	for _, raw := range []string{"https://EXAMPLE.com/", "http://127.0.0.1:8080", "http://10.10.20.123:3003"} {
		if _, err := NormalizeOrigin(raw, true); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := NormalizeOrigin("http://127.0.0.1", false); err == nil {
		t.Fatal("implicit HTTP allowed")
	}
	for _, path := range []string{"https://evil.test/api/v1/test", "//evil.test/api/v1/test", "/api/v1/../../secret", "/api/v1/%2e%2e/%2e%2e/secret", "/config.js", "/api/v1/evil\\secret"} {
		if _, err := ResourceURL(DefaultOrigin, path); err == nil {
			t.Fatalf("accepted unsafe path %s", path)
		}
	}
	if ProfileID(DefaultOrigin) == ProfileID("https://other.test") {
		t.Fatal("profile collision")
	}
}

func relayRequest(t *testing.T, r *Relay, method, path, cap, origin string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, r.URL+path, strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(CapabilityHeader, cap)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestRelayAuthorizationAndScopes(t *testing.T) {
	paths := []string{"/files?file_path=resource%3A%2F%2Ftest", "/api/v1/knowledge-bases/kb/files", "/api/v1/sessions/s/messages/m/files", "/api/v1/embed/channel/files"}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(CapabilityHeader) != "" || r.URL.Query().Get(wsCapabilityQuery) != "" {
			t.Error("local capability leaked")
		}
		if !strings.HasPrefix(r.Host, "127.0.0.1:") {
			t.Errorf("incorrect cloud Host: %s", r.Host)
		}
		if r.Header.Get("Cookie") != "" {
			t.Error("cookie forwarded")
		}
		if r.Header.Get("Authorization") != "Bearer cloud-token" || r.Header.Get("X-Tenant-ID") != "42" {
			t.Error("auth scope lost")
		}
		w.Header().Set("Access-Control-Allow-Origin", "https://wrong.test")
		w.Header().Set("Content-Disposition", "attachment; filename=test.txt")
		w.WriteHeader(403)
		fmt.Fprint(w, "cloud permission denied")
	}))
	defer upstream.Close()
	r, err := NewRelay(upstream.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, origin := range []string{"wails://wails", "http://wails.localhost", "null"} {
		resp := relayRequest(t, r, "OPTIONS", "/api/v1/test", "", origin)
		resp.Body.Close()
		if resp.StatusCode != 204 || resp.Header.Get("Access-Control-Allow-Origin") != origin || !strings.Contains(resp.Header.Get("Access-Control-Allow-Headers"), "X-Embed-Session") {
			t.Fatal("native/embed preflight rejected")
		}
	}
	for _, tc := range []struct {
		cap, origin, path string
		status            int
	}{
		{"", "wails://wails.localhost", "/api/v1/test", 403},
		{r.Capability, "https://evil.test", "/api/v1/test", 403},
		{r.Capability, "wails://wails.localhost", "/preferences.json", 404},
		{r.Capability, "wails://wails.localhost", "/api/v1/../secret", 404},
	} {
		resp := relayRequest(t, r, "POST", tc.path, tc.cap, tc.origin)
		resp.Body.Close()
		if resp.StatusCode != tc.status {
			t.Fatalf("%s got %d", tc.path, resp.StatusCode)
		}
	}
	for _, path := range paths {
		req, _ := http.NewRequest("POST", r.URL+path, strings.NewReader("payload"))
		req.Header.Set(CapabilityHeader, r.Capability)
		req.Header.Set("Origin", "wails://wails.localhost")
		req.Header.Set("Authorization", "Bearer cloud-token")
		req.Header.Set("X-Tenant-ID", "42")
		req.Header.Set("Cookie", "secret=cookie")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 403 || string(data) != "cloud permission denied" {
			t.Fatal("cloud status/body changed")
		}
		if resp.Header.Get("Access-Control-Allow-Origin") != "wails://wails.localhost" {
			t.Fatal("local CORS overwritten")
		}
		if resp.Header.Get("Content-Disposition") == "" {
			t.Fatal("download headers lost")
		}
	}
}

func TestSSEFlushAndCancellation(t *testing.T) {
	canceled := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(canceled)
	}))
	defer upstream.Close()
	r, err := NewRelay(upstream.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", r.URL+"/api/v1/stream", nil)
	req.Header.Set(CapabilityHeader, r.Capability)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil || line != "data: first\n" {
		t.Fatalf("stream was buffered: %q %v", line, err)
	}
	cancel()
	resp.Body.Close()
	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel did not reach upstream")
	}
}

func TestWebSocketTicketAndShutdown(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == "http://"+r.Host }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ticket") != "cloud-ticket" || r.URL.Query().Get(wsCapabilityQuery) != "" {
			t.Error("WS query scope changed")
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		kind, data, err := c.ReadMessage()
		if err == nil {
			_ = c.WriteMessage(kind, data)
		}
		_, _, _ = c.ReadMessage()
	}))
	defer upstream.Close()
	r, err := NewRelay(upstream.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	u := strings.Replace(r.URL, "http:", "ws:", 1) + "/api/v1/terminal?ticket=cloud-ticket&" + wsCapabilityQuery + "=" + r.Capability
	h := http.Header{"Origin": []string{"wails://wails.localhost"}}
	c, _, err := websocket.DefaultDialer.Dial(u, h)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err = c.WriteMessage(websocket.TextMessage, []byte("echo")); err != nil {
		t.Fatal(err)
	}
	_, data, err := c.ReadMessage()
	if err != nil || string(data) != "echo" {
		t.Fatalf("WS relay failed %s %v", data, err)
	}
	r.Close()
	_ = c.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err = c.ReadMessage(); err == nil {
		t.Fatal("socket stayed open on shutdown")
	}
}

func TestDownloadDoesNotLeakAuthOnRedirect(t *testing.T) {
	called := false
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; t.Error("foreign redirect followed") }))
	defer other.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("auth missing")
		}
		http.Redirect(w, r, other.URL+"/api/v1/file", 302)
	}))
	defer upstream.Close()
	if _, err := Download(context.Background(), upstream.URL, "/api/v1/download", "secret", "42", ""); err == nil {
		t.Fatal("unsafe redirect allowed")
	}
	if called {
		t.Fatal("foreign server received request")
	}
}

func TestMultipartAndEmbedAuthUnchanged(t *testing.T) {
	const body = "--boundary\r\nContent-Disposition: form-data; name=\"file\"; filename=\"test.txt\"\r\n\r\npayload\r\n--boundary--\r\n"
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if string(data) != body || r.Header.Get("Content-Type") != "multipart/form-data; boundary=boundary" || r.Header.Get("Authorization") != "Embed guest" {
			t.Error("upload or embed auth changed")
		}
		w.WriteHeader(201)
	}))
	defer up.Close()
	r, err := NewRelay(up.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	req, _ := http.NewRequest("POST", r.URL+"/api/v1/embed/c/upload", strings.NewReader(body))
	req.Header.Set(CapabilityHeader, r.Capability)
	req.Header.Set("Authorization", "Embed guest")
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatal(resp.StatusCode)
	}
}
