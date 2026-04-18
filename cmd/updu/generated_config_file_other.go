//go:build !unix

package main

import "os"

func createGeneratedConfigFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
}
