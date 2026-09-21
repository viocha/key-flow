//go:build windows

package main

import "testing"

func TestStartupTaskQuery(t *testing.T) {
	if _, err := startupTaskEnabled(); err != nil {
		t.Fatal(err)
	}
}
