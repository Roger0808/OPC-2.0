//go:build windows

package securestore

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"unsafe"
)

type dataBlob struct {
	Size uint32
	Data *byte
}

var crypt32 = windows.NewLazySystemDLL("crypt32.dll")
var protect = crypt32.NewProc("CryptProtectData")
var unprotect = crypt32.NewProc("CryptUnprotectData")

func transform(proc *windows.LazyProc, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	in := dataBlob{uint32(len(data)), &data[0]}
	var out dataBlob
	// CRYPTPROTECT_UI_FORBIDDEN, current Windows user. Secrets never appear in argv.
	ok, _, err := proc.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&out)))
	if ok == 0 {
		return nil, fmt.Errorf("Windows 凭据加密失败: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
func secretPath(dir, profile string) string { return filepath.Join(dir, "session-"+profile+".dpapi") }
func loadSecret(dir, profile string) ([]byte, error) {
	data, err := os.ReadFile(secretPath(dir, profile))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return transform(unprotect, data)
}
func saveSecret(dir, profile string, data []byte) error {
	encoded, err := transform(protect, data)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".session-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(encoded); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	// Windows Rename does not overwrite an existing file; MoveFileEx is atomic replacement.
	src, _ := windows.UTF16PtrFromString(tmp.Name())
	dest, _ := windows.UTF16PtrFromString(secretPath(dir, profile))
	return windows.MoveFileEx(src, dest, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
func deleteSecret(dir, profile string) error {
	err := os.Remove(secretPath(dir, profile))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
