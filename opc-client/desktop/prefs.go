package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type desktopPrefs struct {
	ServerURL string `json:"server_url"`
}

func desktopPrefsDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	if isolated := os.Getenv("OPC_CLIENT_CONFIG_DIR"); isolated != "" {
		base = isolated
	}
	dir := filepath.Join(base, "OPC Client")
	return dir, os.MkdirAll(dir, 0700)
}
func loadDesktopPrefs(dir string) desktopPrefs {
	data, err := os.ReadFile(filepath.Join(dir, "preferences.json"))
	if err != nil {
		return desktopPrefs{}
	}
	var p desktopPrefs
	_ = json.Unmarshal(data, &p)
	return p
}
func saveDesktopPrefs(dir string, p desktopPrefs) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".preferences-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return replaceFile(f.Name(), filepath.Join(dir, "preferences.json"))
}
