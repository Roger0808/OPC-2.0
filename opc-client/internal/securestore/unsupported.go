//go:build (!darwin && !windows) || (darwin && !cgo)

package securestore

import "fmt"

func loadSecret(_, _ string) ([]byte, error) {
	return nil, fmt.Errorf("当前构建不支持系统凭据存储")
}
func saveSecret(_, _ string, _ []byte) error {
	return fmt.Errorf("当前构建不支持系统凭据存储")
}
func deleteSecret(_, _ string) error { return nil }
