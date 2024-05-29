package main

import "path/filepath"

func LocalX3fExtractPath() string {
	return filepath.Join(MustUserHomeDir(), ".local", "bin", "x3f_extract")
}
