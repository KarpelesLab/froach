package froach

import (
	"os"
	"path/filepath"
)

func basePath() string {
	p, err := os.UserConfigDir()
	if err != nil {
		p = "/tmp"
	}
	return filepath.Join(p, "froach")
}

func cachePath() string {
	p, err := os.UserCacheDir()
	if err != nil {
		p = "/tmp"
	}
	return filepath.Join(p, "froach")
}
