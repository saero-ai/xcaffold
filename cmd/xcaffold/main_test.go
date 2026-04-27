package main

import "os"

func init() {
	os.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
}
