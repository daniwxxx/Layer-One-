package storage

import (
	"fmt"
	"os"
	"time"
)

type fileLock struct {
	path string
}

const (
	lockRetryInterval = 50 * time.Millisecond
	lockTimeout       = 5 * time.Second
	lockStaleAfter    = 30 * time.Second
)

func acquireFileLock(dbPath string) (*fileLock, error) {
	lockPath := dbPath + ".lock"
	deadline := time.Now().Add(lockTimeout)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			_ = f.Close()
			return &fileLock{path: lockPath}, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
		if info, statErr := os.Stat(lockPath); statErr == nil {
			if time.Since(info.ModTime()) > lockStaleAfter {
				_ = os.Remove(lockPath)
				continue
			}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("no se pudo tomar el lock de %s", dbPath)
		}
		time.Sleep(lockRetryInterval)
	}
}

func (l *fileLock) release() {
	if l == nil {
		return
	}
	_ = os.Remove(l.path)
}
