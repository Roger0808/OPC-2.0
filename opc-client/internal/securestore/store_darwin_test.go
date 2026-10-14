//go:build darwin && cgo

package securestore

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMacOSKeychainRoundTrip(t *testing.T) {
	if os.Getenv("OPC_TEST_KEYCHAIN") != "1" {
		t.Skip("explicit OS keychain integration check")
	}
	profile := fmt.Sprintf("qa-%d", time.Now().UnixNano())
	s := New(t.TempDir())
	defer deleteSecret("", profile)
	want := Session{Token: "qa-access-token", RefreshToken: "qa-refresh-token"}
	if err := s.Save(profile, want); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load(profile)
	if err != nil || got != want {
		t.Fatalf("keychain round trip failed: %v", err)
	}
	if err := s.Save(profile, Session{}); err != nil {
		t.Fatal(err)
	}
	got, err = s.Load(profile)
	if err != nil || got != (Session{}) {
		t.Fatal("logout restored credentials")
	}
	if err := s.Save(profile, want); err != nil {
		t.Fatal(err)
	}
	got, err = s.Load(profile)
	if err != nil || got != want {
		t.Fatal("login after revocation failed")
	}
}
