package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const currentVersion = 1

type Record struct {
	Provider   string `json:"provider"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	PublicIP   string `json:"public_ip,omitempty"`
	Game       string `json:"game,omitempty"`
	ConfigPath string `json:"config_path,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type Snapshot struct {
	Version int      `json:"version"`
	Servers []Record `json:"servers"`
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func DefaultStore() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}

	return NewStore(path), nil
}

func DefaultPath() (string, error) {
	path, err := resolveDefaultStatePath()
	if err != nil {
		return "", fmt.Errorf("resolve default state path: %w", err)
	}

	return path, nil
}

func (s *Store) Load() (*Snapshot, error) {
	if s == nil {
		return nil, fmt.Errorf("state store is nil")
	}
	if s.path == "" {
		return nil, fmt.Errorf("state store path is empty")
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Snapshot{Version: currentVersion, Servers: []Record{}}, nil
		}
		return nil, fmt.Errorf("read state file %q: %w", s.path, err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("decode state file %q: %w", s.path, err)
	}

	if snapshot.Version == 0 {
		snapshot.Version = currentVersion
	}
	if snapshot.Servers == nil {
		snapshot.Servers = []Record{}
	}

	return &snapshot, nil
}

func (s *Store) SaveRecord(record Record) error {
	if s == nil {
		return fmt.Errorf("state store is nil")
	}
	if s.path == "" {
		return fmt.Errorf("state store path is empty")
	}
	if record.Provider == "" {
		return fmt.Errorf("state record provider cannot be empty")
	}
	if record.ID == "" {
		return fmt.Errorf("state record id cannot be empty")
	}

	snapshot, err := s.Load()
	if err != nil {
		return err
	}

	replaced := false
	for i := range snapshot.Servers {
		if snapshot.Servers[i].Provider == record.Provider && snapshot.Servers[i].ID == record.ID {
			snapshot.Servers[i] = record
			replaced = true
			break
		}
	}
	if !replaced {
		snapshot.Servers = append(snapshot.Servers, record)
	}

	snapshot.Version = currentVersion
	return s.writeSnapshot(*snapshot)
}

func (s *Store) writeSnapshot(snapshot Snapshot) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create state directory %q: %w", dir, err)
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state snapshot: %w", err)
	}
	data = append(data, '\n')

	tmpFile, err := os.CreateTemp(dir, "instances-*.json")
	if err != nil {
		return fmt.Errorf("create temp state file in %q: %w", dir, err)
	}

	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp state file %q: %w", tmpPath, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp state file %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace state file %q: %w", s.path, err)
	}

	return nil
}
