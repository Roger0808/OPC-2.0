package cloud

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/url"
	"strings"
)

const DefaultOrigin = "https://admin.tokendance.sh"

// Config keeps public URLs separate from the ephemeral local transport.
type Config struct {
	PublicOrigin    string `json:"publicOrigin"`
	ProfileID       string `json:"profileId"`
	TransportBase   string `json:"transportBase"`
	RelayCapability string `json:"relayCapability"`
	RemoteDesktop   bool   `json:"remoteDesktop"`
	Version         string `json:"version"`
}

// NormalizeOrigin accepts a single server origin, never credentials or an API path.
// HTTP is available only for explicit development on loopback/private addresses.
func NormalizeOrigin(raw string, allowHTTP bool) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", fmt.Errorf("请输入服务端地址（例如 https://admin.tokendance.sh），不要包含 API 路径或登录信息")
	}
	if u.Scheme != "https" {
		ip := net.ParseIP(u.Hostname())
		private := strings.EqualFold(u.Hostname(), "localhost") || (ip != nil && (ip.IsLoopback() || ip.IsPrivate()))
		if u.Scheme != "http" || !allowHTTP || !private {
			return "", fmt.Errorf("服务端连接必须使用 HTTPS；本地开发 HTTP 需要显式启用")
		}
	}
	u.Host = strings.ToLower(u.Host)
	u.Path, u.RawPath = "", ""
	return u.String(), nil
}

func ProfileID(origin string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(origin)))[:24] }

func AllowedPath(path string) bool {
	if strings.Contains(path, "\\") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return strings.HasPrefix(path, "/api/v1/") || path == "/files" || strings.HasPrefix(path, "/files/") || strings.HasPrefix(path, "/mcp/") || strings.HasPrefix(path, "/r/")
}

func ResourceURL(origin, path string) (*url.URL, error) {
	u, err := url.Parse(path)
	if err != nil || u.IsAbs() || u.Host != "" || u.User != nil || u.Fragment != "" || !AllowedPath(u.Path) || strings.Contains(u.Path, "\\") {
		return nil, fmt.Errorf("不支持的资源地址")
	}
	base, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}
	return base.ResolveReference(u), nil
}
