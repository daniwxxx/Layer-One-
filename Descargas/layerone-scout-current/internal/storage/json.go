package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"layerone-scout/internal/model"
)

const currentSchemaVersion = 1

type JSONStore struct {
	mu        sync.Mutex
	path      string
	backupDir string
}

func NewJSONStore(path, backupDir string) (*JSONStore, error) {
	return &JSONStore{path: path, backupDir: backupDir}, nil
}

func (s *JSONStore) Load() (*Database, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *JSONStore) loadLocked() (*Database, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.newDatabase(), nil
		}
		return nil, err
	}
	if len(b) == 0 {
		return s.newDatabase(), nil
	}
	var db Database
	if err := json.Unmarshal(b, &db); err != nil {
		backup, err := s.restoreFromBackup()
		if err == nil {
			return backup, nil
		}
		return nil, fmt.Errorf("corrupto y sin backup: %w", err)
	}
	if db.SchemaVersion == 0 {
		db.SchemaVersion = 1
	}
	return &db, nil
}

func (s *JSONStore) Save(db *Database) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock, err := acquireFileLock(s.path)
	if err != nil {
		return err
	}
	defer lock.release()
	return s.saveLocked(db)
}

func (s *JSONStore) saveLocked(db *Database) error {
	if err := s.backupLocked(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".scout-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}

func (s *JSONStore) Mutate(fn func(*Database) error) (*Database, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock, err := acquireFileLock(s.path)
	if err != nil {
		return nil, err
	}
	defer lock.release()
	db, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	if err := fn(db); err != nil {
		return nil, err
	}
	db.UpdatedAt = time.Now()
	if err := s.saveLocked(db); err != nil {
		return nil, err
	}
	return db, nil
}

func (s *JSONStore) Close() error { return nil }

func (s *JSONStore) newDatabase() *Database {
	now := time.Now()
	return &Database{
		SchemaVersion: currentSchemaVersion,
		Persons:       []model.Person{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
