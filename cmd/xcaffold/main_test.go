package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	tmpHome, err := os.MkdirTemp("", "xcaffold-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpHome)
	os.Setenv("HOME", tmpHome)
	os.Exit(m.Run())
}
