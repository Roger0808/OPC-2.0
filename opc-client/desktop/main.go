//go:build !bindings

package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	wr "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:assets
var bundled embed.FS

func main() {
	app := NewApp()
	if err := app.initialize(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer app.relay.Close()
	assets, err := fs.Sub(bundled, "assets")
	if err != nil {
		panic(err)
	}
	appMenu := menu.NewMenu()
	fileMenu := appMenu.AddSubmenu("OPC Client")
	fileMenu.AddText("服务端连接…", nil, func(_ *menu.CallbackData) {
		wr.WindowExecJS(app.ctx, "window.dispatchEvent(new Event('opc:connection-settings'))")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("退出", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) { wr.Quit(app.ctx) })
	appMenu.Append(menu.EditMenu())
	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Reload", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) { wr.WindowReload(app.ctx) })
	cfg, _ := json.Marshal(app.config)
	runtimeJS := "window.__OPC_CLIENT_RUNTIME__=" + string(cfg) + ";window.__RUNTIME_CONFIG__={MAX_FILE_SIZE_MB:50,MAX_SKILL_BUNDLE_SIZE_MB:256};"
	err = wails.Run(&options.App{
		Title: "OPC Client", Width: 1280, Height: 800, Menu: appMenu,
		AssetServer: &assetserver.Options{
			Assets: assets,
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/config.js" {
						w.Header().Set("Content-Type", "application/javascript")
						w.Header().Set("Cache-Control", "no-store")
						fmt.Fprint(w, runtimeJS)
						return
					}
					next.ServeHTTP(w, r)
				})
			},
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/files/") {
					http.NotFound(w, r)
					return
				}
				data, e := fs.ReadFile(assets, "index.html")
				if e != nil {
					http.Error(w, "Frontend is not built", 503)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(data)
			}),
		},
		OnStartup: app.startup, OnShutdown: app.shutdown,
		OnDomReady: func(ctx context.Context) { wr.WindowExecJS(ctx, wailsThemeSyncJS); wr.WindowExecJS(ctx, dragHandlerJS) },
		Bind:       []interface{}{app}, BackgroundColour: &options.RGBA{R: 238, G: 238, B: 238, A: 255},
		Mac: &mac.Options{TitleBar: mac.TitleBarHiddenInset(), WebviewIsTransparent: false, WindowIsTranslucent: false},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
