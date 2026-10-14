package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Tencent/WeKnora/opc-client/internal/cloud"
	"github.com/Tencent/WeKnora/opc-client/internal/securestore"
	wr "github.com/wailsapp/wails/v2/pkg/runtime"
)

var version = "0.1.0"

type App struct {
	ctx            context.Context
	config         cloud.Config
	relay          *cloud.Relay
	credentials    *securestore.Store
	prefsDir       string
	saving         atomic.Bool
	sessionMu      sync.Mutex
	session        securestore.Session
	sessionLoaded  bool
	downloadMu     sync.Mutex
	downloadCancel context.CancelFunc
}

func NewApp() *App { return &App{} }
func (a *App) initialize() error {
	dir, err := desktopPrefsDir()
	if err != nil {
		return err
	}
	a.prefsDir = dir
	origin := loadDesktopPrefs(dir).ServerURL
	if env := os.Getenv("OPC_CLIENT_SERVER_URL"); env != "" {
		origin = env
	}
	if origin == "" {
		origin = cloud.DefaultOrigin
	}
	origin, err = cloud.NormalizeOrigin(origin, os.Getenv("OPC_CLIENT_ALLOW_HTTP") == "1")
	if err != nil {
		return err
	}
	relay, err := cloud.NewRelay(origin, os.Getenv("OPC_CLIENT_DEV_ORIGIN"))
	if err != nil {
		return err
	}
	a.relay = relay
	a.credentials = securestore.New(dir)
	a.config = cloud.Config{PublicOrigin: origin, ProfileID: cloud.ProfileID(origin), TransportBase: relay.URL, RelayCapability: relay.Capability, RemoteDesktop: true, Version: version}
	return nil
}
func (a *App) startup(ctx context.Context)       { a.ctx = ctx }
func (a *App) shutdown(_ context.Context)        { a.relay.Close() }
func (a *App) GetConnectionConfig() cloud.Config { return a.config }
func (a *App) GetAPIBaseURL() string             { return a.config.PublicOrigin + "/api/v1" }
func (a *App) SetServerURL(raw string) error {
	origin, err := cloud.NormalizeOrigin(raw, os.Getenv("OPC_CLIENT_ALLOW_HTTP") == "1")
	if err != nil {
		return err
	}
	// Applied only on next launch: no requests from an old account can cross servers.
	return saveDesktopPrefs(a.prefsDir, desktopPrefs{ServerURL: origin})
}
func (a *App) GetSavedSession() (securestore.Session, error) {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	if a.sessionLoaded {
		return a.session, nil
	}
	value, err := a.credentials.Load(a.config.ProfileID)
	if err == nil {
		a.session = value
		a.sessionLoaded = true
	}
	return value, err
}
func (a *App) SaveSession(profile string, session securestore.Session) error {
	if profile != a.config.ProfileID {
		return fmt.Errorf("session profile mismatch")
	}
	if len(session.Token) > 64<<10 || len(session.RefreshToken) > 64<<10 {
		return fmt.Errorf("invalid credential size")
	}
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	a.session = session
	a.sessionLoaded = true
	return a.credentials.Save(profile, session)
}

// DownloadFile streams a protected cloud resource to a user-selected destination.
// No arbitrary server URL or silently chosen local write path is accepted.
func (a *App) DownloadFile(path, name, tenant, embedToken string) (bool, error) {
	return a.DownloadResource(path, name, tenant, embedToken, "GET", "")
}

func (a *App) CancelDownload() {
	a.downloadMu.Lock()
	defer a.downloadMu.Unlock()
	if a.downloadCancel != nil {
		a.downloadCancel()
	}
}

func (a *App) DownloadResource(path, name, tenant, embedToken, method, body string) (bool, error) {
	if !a.saving.CompareAndSwap(false, true) {
		return false, fmt.Errorf("另一个保存窗口正在打开")
	}
	defer a.saving.Store(false)
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Minute)
	a.downloadMu.Lock()
	a.downloadCancel = cancel
	a.downloadMu.Unlock()
	defer func() { cancel(); a.downloadMu.Lock(); a.downloadCancel = nil; a.downloadMu.Unlock() }()
	if _, err := cloud.ResourceURL(a.config.PublicOrigin, path); err != nil {
		return false, err
	}
	session, err := a.GetSavedSession()
	if err != nil {
		return false, err
	}
	target, err := wr.SaveFileDialog(a.ctx, wr.SaveDialogOptions{Title: "保存文件", DefaultFilename: filepath.Base(name)})
	if err != nil || target == "" {
		return false, err
	}
	resp, err := cloud.DownloadRequest(ctx, a.config.PublicOrigin, path, session.Token, tenant, embedToken, method, body)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	temp, err := os.CreateTemp(filepath.Dir(target), ".opc-download-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(temp.Name())
	_, err = io.Copy(temp, resp.Body)
	closeErr := temp.Close()
	if err != nil {
		return false, err
	}
	if closeErr != nil {
		return false, closeErr
	}
	if err = replaceFile(temp.Name(), target); err != nil {
		return false, err
	}
	return true, nil
}

// SaveData covers generated exports and already-loaded previews (bounded to 64 MiB).
func (a *App) SaveData(name string, encoded string) (bool, error) {
	if len(encoded) > (64<<20)*4/3+4 {
		return false, fmt.Errorf("文件过大，请使用服务端流式下载")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false, fmt.Errorf("无效的文件数据")
	}
	if len(data) > 64<<20 {
		return false, fmt.Errorf("文件过大，请使用服务端流式下载")
	}
	if !a.saving.CompareAndSwap(false, true) {
		return false, fmt.Errorf("另一个保存窗口正在打开")
	}
	defer a.saving.Store(false)
	target, err := wr.SaveFileDialog(a.ctx, wr.SaveDialogOptions{Title: "保存文件", DefaultFilename: filepath.Base(strings.TrimSpace(name))})
	if err != nil || target == "" {
		return false, err
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".opc-export-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(temp.Name())
	_, err = temp.Write(data)
	closeErr := temp.Close()
	if err != nil {
		return false, err
	}
	if closeErr != nil {
		return false, closeErr
	}
	if err = replaceFile(temp.Name(), target); err != nil {
		return false, err
	}
	return true, nil
}
