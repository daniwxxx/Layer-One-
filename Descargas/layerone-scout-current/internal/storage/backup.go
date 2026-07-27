package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *JSONStore) backupLocked() error {
	info, err := os.Stat(s.path)
	if err != nil || info.Size() == 0 {
		return nil
	}
	backupDir := s.effectiveBackupDir()
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}
	backupName := fmt.Sprintf("%s.%s.bak", filepath.Base(s.path), time.Now().Format("20060102-150405"))
	backupPath := filepath.Join(backupDir, backupName)
	src, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

func (s *JSONStore) restoreFromBackup() (*Database, error) {
	backupDir := s.effectiveBackupDir()
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}
	var latest string
	var latestTime time.Time
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".bak") {
			info, err := e.Info()
			if err == nil && info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latest = filepath.Join(backupDir, e.Name())
			}
		}
	}
	if latest == "" {
		return nil, fmt.Errorf("no hay backup disponible")
	}
	b, err := os.ReadFile(latest)
	if err != nil {
		return nil, err
	}
	var db Database
	if err := json.Unmarshal(b, &db); err != nil {
		return nil, err
	}
	return &db, nil
}

func (s *JSONStore) effectiveBackupDir() string {
	if s.backupDir != "" {
		return s.backupDir
	}
	if s.path == "" {
		return "backups"
	}
	return filepath.Join(filepath.Dir(s.path), "backups")
}
