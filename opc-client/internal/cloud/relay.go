package cloud

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const CapabilityHeader = "X-OPC-Client-Session"
const wsCapabilityQuery = "opc_client_session"

type Relay struct {
	URL         string
	Capability  string
	server      *http.Server
	transport   *http.Transport
	mu          sync.Mutex
	connections map[net.Conn]struct{}
}

func NewRelay(origin string, developmentOrigin string) (*Relay, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid upstream origin")
	}
	random := make([]byte, 32)
	if _, err = rand.Read(random); err != nil {
		return nil, err
	}
	r := &Relay{Capability: hex.EncodeToString(random), connections: make(map[net.Conn]struct{})}
	r.transport = http.DefaultTransport.(*http.Transport).Clone()
	dial := r.transport.DialContext
	r.transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		t := &trackedConnection{Conn: conn, relay: r}
		r.mu.Lock()
		r.connections[t] = struct{}{}
		r.mu.Unlock()
		return t, nil
	}
	r.transport.ResponseHeaderTimeout = 0 // SSE and long-running tool handshakes.
	r.transport.TLSHandshakeTimeout = 15 * time.Second
	proxy := &httputil.ReverseProxy{
		Transport:     r.transport,
		FlushInterval: -1,
		Rewrite: func(p *httputil.ProxyRequest) {
			p.SetURL(u) // Includes the remote Host; cloud routing must not see wails.localhost.
			p.Out.Header.Del(CapabilityHeader)
			p.Out.Header.Del("Cookie") // JWT/Embed auth only; native helper has no cookie jar.
			p.Out.Header.Del("Referer")
			p.Out.Header.Set("Origin", origin)
			q := p.Out.URL.Query()
			q.Del(wsCapabilityQuery)
			p.Out.URL.RawQuery = q.Encode()
		},
		ModifyResponse: func(resp *http.Response) error {
			// CORS belongs to this local relay, not the server's browser policy.
			for k := range resp.Header {
				if strings.HasPrefix(strings.ToLower(k), "access-control-") {
					resp.Header.Del(k)
				}
			}
			resp.Header.Del("Set-Cookie")
			// API redirects are followed only when they remain on the selected server.
			if loc := resp.Header.Get("Location"); loc != "" {
				dest, e := resp.Request.URL.Parse(loc)
				if e != nil || dest.Scheme != u.Scheme || dest.Host != u.Host || !AllowedPath(dest.Path) {
					return fmt.Errorf("unsafe upstream redirect")
				}
				resp.Header.Set("Location", r.URL+dest.RequestURI())
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"success":false,"message":"无法连接当前服务端，请检查网络后重试"}`))
		},
	}
	allowedOrigins := map[string]bool{"wails://wails": true, "wails://wails.localhost": true, "http://wails.localhost": true, "https://wails.localhost": true, "null": true}
	if developmentOrigin != "" {
		allowedOrigins[developmentOrigin] = true
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Host != strings.TrimPrefix(r.URL, "http://") {
			http.Error(w, "invalid host", 403)
			return
		}
		o := req.Header.Get("Origin")
		if o != "" && !allowedOrigins[o] {
			http.Error(w, "invalid origin", 403)
			return
		}
		if o != "" {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept-Language, X-Tenant-ID, X-Request-ID, X-OPC-Client-Session, X-Embed-Token, X-Embed-Session, X-Embed-Visitor, X-API-Key, X-External-User-ID, X-External-User-Token, Cache-Control, Last-Event-ID, Range")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition, Content-Length, Content-Range")
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
		}
		if !AllowedPath(req.URL.Path) {
			http.NotFound(w, req)
			return
		}
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		capability := req.Header.Get(CapabilityHeader)
		if strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
			capability = req.URL.Query().Get(wsCapabilityQuery)
		}
		if subtle.ConstantTimeCompare([]byte(capability), []byte(r.Capability)) != 1 {
			http.Error(w, "missing capability", 403)
			return
		}
		proxy.ServeHTTP(w, req)
	})
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		r.transport.CloseIdleConnections()
		return nil, err
	}
	r.URL = "http://" + listener.Addr().String()
	r.server = &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second, MaxHeaderBytes: 64 << 10}
	go func() { _ = r.server.Serve(listener) }()
	return r, nil
}

func (r *Relay) Close() {
	if r != nil {
		_ = r.server.Close()
		r.transport.CloseIdleConnections()
		r.mu.Lock()
		conns := make([]net.Conn, 0, len(r.connections))
		for c := range r.connections {
			conns = append(conns, c)
		}
		r.mu.Unlock()
		for _, c := range conns {
			_ = c.Close()
		} // Includes hijacked WebSocket upstream connections.
	}
}

type trackedConnection struct {
	net.Conn
	relay *Relay
}

func (c *trackedConnection) Close() error {
	c.relay.mu.Lock()
	delete(c.relay.connections, c)
	c.relay.mu.Unlock()
	return c.Conn.Close()
}

// Download performs a bounded-target request without buffering the response body.
func Download(ctx context.Context, origin, path, token, tenant, embedToken string) (*http.Response, error) {
	return DownloadRequest(ctx, origin, path, token, tenant, embedToken, http.MethodGet, "")
}

var batchDownloadPath = regexp.MustCompile(`^/api/v1/knowledge-bases/[^/]+/knowledge/batch-download$`)

func DownloadRequest(ctx context.Context, origin, path, token, tenant, embedToken, method, body string) (*http.Response, error) {
	u, err := ResourceURL(origin, path)
	if err != nil {
		return nil, err
	}
	if method != "GET" && (method != "POST" || !batchDownloadPath.MatchString(u.Path) || len(body) > 2<<20) {
		return nil, fmt.Errorf("不支持的下载请求")
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	if embedToken != "" {
		req.Header.Set("Authorization", "Embed "+embedToken)
	} else if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if tenant != "" && embedToken == "" {
		req.Header.Set("X-Tenant-ID", tenant)
	}
	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || req.URL.Scheme != u.Scheme || req.URL.Host != u.Host || !AllowedPath(req.URL.Path) {
			return fmt.Errorf("下载重定向超出当前服务端")
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("文件下载失败（HTTP %d）", resp.StatusCode)
	}
	return resp, nil
}
