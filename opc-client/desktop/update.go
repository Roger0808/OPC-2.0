package main

import wr "github.com/wailsapp/wails/v2/pkg/runtime"

// No OPC update feed has been configured. Never query the upstream Lite feed.
func (a *App) AutoCheckForUpdates() {}
func (a *App) CheckForUpdates() {
	if a.ctx != nil {
		_, _ = wr.MessageDialog(a.ctx, wr.MessageDialogOptions{Type: wr.InfoDialog, Title: "OPC Client", Message: "此版本尚未配置 OPC 客户端更新源，请使用管理员提供的安装包更新。"})
	}
}
