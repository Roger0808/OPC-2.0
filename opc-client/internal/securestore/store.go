package securestore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Session struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

// Store serializes access to the OS credential store. No plaintext disk fallback.
type Store struct {
	mu  sync.Mutex
	dir string
}

func New(dir string) *Store { return &Store{dir: dir} }
func (s *Store) Load(profile string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Even when the OS store is locked during logout, never resurrect an older session.
	if _, err := os.Stat(s.revocationPath(profile)); err == nil {
		return Session{}, nil
	} else if !os.IsNotExist(err) {
		return Session{}, err
	}
	data, err := loadSecret(s.dir, profile)
	if err != nil || len(data) == 0 {
		return Session{}, err
	}
	var session Session
	err = json.Unmarshal(data, &session)
	return session, err
}
func (s *Store) Save(profile string, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session.Token == "" && session.RefreshToken == "" {
		if err := os.MkdirAll(s.dir, 0700); err != nil {
			return err
		}
		if err := os.WriteFile(s.revocationPath(profile), []byte("logged-out\n"), 0600); err != nil {
			return err
		}
		return deleteSecret(s.dir, profile)
	}
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	if err = saveSecret(s.dir, profile, data); err != nil {
		return err
	}
	if err = os.Remove(s.revocationPath(profile)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Store) revocationPath(profile string) string {
	return filepath.Join(s.dir, "session-"+profile+".revoked")
}
