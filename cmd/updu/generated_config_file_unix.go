//go:build unix

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func createGeneratedConfigFile(path string) (*os.File, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve output path: %w", err)
	}

	parentFD, err := openNoFollowDirectory(filepath.Dir(absPath))
	if err != nil {
		return nil, err
	}
	defer unix.Close(parentFD)

	fd, err := unix.Openat(parentFD, filepath.Base(absPath), unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), absPath), nil
}

func openNoFollowDirectory(absDir string) (int, error) {
	fd, err := unix.Open(string(os.PathSeparator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, fmt.Errorf("open root directory: %w", err)
	}

	trimmed := strings.TrimPrefix(absDir, string(os.PathSeparator))
	if trimmed == "" {
		return fd, nil
	}

	currentFD := fd
	for _, component := range strings.Split(trimmed, string(os.PathSeparator)) {
		nextFD, err := unix.Openat(currentFD, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		unix.Close(currentFD)
		if err != nil {
			return -1, err
		}
		currentFD = nextFD
	}

	return currentFD, nil
}
