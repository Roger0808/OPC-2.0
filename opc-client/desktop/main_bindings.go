//go:build bindings

package main

import (
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

// Wails generates bindings with this entry point; the cloud relay and UI are not started.
func main() {
	app := NewApp()
	_ = wails.Run(&options.App{
		Title: "OPC Client",
		Bind:  []interface{}{app},
	})
}
