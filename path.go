package froach

import (
	"os"
	"path/filepath"
)

var (
	BasePath  = initialBasePath()
	CachePath = initialCachePath()
)

func cachePath() string {
	return CachePath
}

func basePath() string {
	return BasePath
}

func initialBasePath() string {
	p, err := os.UserConfigDir()
	if err != nil {
		p = "/tmp"
	}
	return filepath.Join(p, "froach")
}

func initialCachePath() string {
	p, err := os.UserCacheDir()
	if err != nil {
		p = "/tmp"
	}
	return filepath.Join(p, "froach")
}
