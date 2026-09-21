//go:build windows

package main

import "os"

func replaceFile(source, target string) error {
	_ = os.Remove(target)
	return os.Rename(source, target)
}
