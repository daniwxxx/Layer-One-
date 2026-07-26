package storage

import (
	"os"
	"syscall"
)

type fileLock struct {
	file *os.File
	path string
}

func acquireFileLock(dbPath string) (*fileLock, error) {
	lockPath := dbPath + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return &fileLock{file: f, path: lockPath}, nil
}

func (l *fileLock) release() {
	if l.file != nil {
		syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
		l.file.Close()
	}
	os.Remove(l.path)
}
