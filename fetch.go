package froach

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/KarpelesLab/fileutil"
	"github.com/KarpelesLab/webutil"
)

type CockroachVersion struct {
	Filename string
	hash     []byte
}

// Exe returns the path to cockroach latest version
func Exe() (string, error) {
	if runtime.GOOS == "linux" {
		// if linux, check if azusa version is available, and return it if it is
		if _, err := os.Stat("/pkg/main/dev-db.cockroach-bin.core/bin/cockroach"); err == nil {
			return "/pkg/main/dev-db.cockroach-bin.core/bin/cockroach", nil
		}
	}

	v, err := GetLatestVersion()
	if err != nil {
		return "", err
	}
	p := cachePath()

	if _, err = os.Stat(filepath.Join(p, v.Dirname())); err == nil {
		// directory already exists, return exe
		return filepath.Join(p, v.Dirname(), "cockroach"), nil
	}

	err = v.ExtractTo(p)
	if err != nil {
		return "", err
	}

	return filepath.Join(p, v.Dirname(), "cockroach"), nil
}

// Dirname returns the directory name expected for the file. Typically cockroachdb
// archive is a directory with the following files:
//
// cockroach
// lib/libgeos.so
// lib/libgeos_c.so
// LICENSE.txt
// THIRD-PARTY-NOTICES.txt
func (v *CockroachVersion) Dirname() string {
	if res, ok := strings.CutSuffix(v.Filename, ".tgz"); ok {
		return res
	}
	return v.Filename
}

// DownloadTo downloads the version of cockroachdb to a file while performing a checksum
func (v *CockroachVersion) DownloadTo(fn string) error {
	fp, err := os.Create(fn + "~")
	if err != nil {
		return err
	}
	defer os.Remove(fn + "~")
	defer fp.Close()

	r, err := webutil.Get("https://binaries.cockroachdb.com/" + v.Filename)
	if err != nil {
		return err
	}
	defer r.Close()

	h := sha256.New()

	_, err = io.Copy(fp, io.TeeReader(r, h))
	if err != nil {
		return err
	}

	fp.Close()

	if v.hash != nil {
		sum := h.Sum(nil)
		if !bytes.Equal(sum, v.hash) {
			return errors.New("cockroachdb download failed: bad hash")
		}
	}

	// all good
	os.Rename(fn+"~", fn)
	return nil
}

// ExtractTo downloads the version of cockroachdb to a directory while performing a checksum
//
// Typically a directory named v.Dirname() will be created there
func (v *CockroachVersion) ExtractTo(dirname string) error {
	r, err := webutil.Get("https://binaries.cockroachdb.com/" + v.Filename)
	if err != nil {
		return err
	}
	defer r.Close()

	h := sha256.New()
	r, err = gzip.NewReader(io.TeeReader(r, h))
	if err != nil {
		return err
	}
	err = fileutil.TarExtract(r, dirname)
	if err != nil {
		// remove anything we may have created
		os.RemoveAll(filepath.Join(dirname, v.Dirname()))
		return err
	}

	if v.hash != nil {
		sum := h.Sum(nil)
		if !bytes.Equal(sum, v.hash) {
			// remove what we just extracted
			os.RemoveAll(filepath.Join(dirname, v.Dirname()))
			return errors.New("cockroachdb download failed: bad hash")
		}
	}

	return nil
}

// GetLatestVersion gathers information on the latest cockroachdb version from cockroach servers
// and returns a CockroachVersion.
func GetLatestVersion() (*CockroachVersion, error) {
	return GetVersion("latest")
}

// GetVersion gathers information on the specified cockroachdb version and returns a CockroachVersion.
func GetVersion(vers string) (*CockroachVersion, error) {
	// https://binaries.cockroachdb.com/cockroach-$vers.linux-amd64.tgz.sha256sum
	u := fmt.Sprintf("https://binaries.cockroachdb.com/cockroach-%s.%s-%s.tgz.sha256sum", vers, runtime.GOOS, runtime.GOARCH)
	nfo, err := readStream(webutil.Get(u))
	if err != nil {
		return nil, err
	}
	nfoA := strings.Fields(string(nfo))
	if len(nfoA) != 2 {
		return nil, fmt.Errorf("unexpected response from server: %s", nfo)
	}

	// nfoA[1] == cockroach-v24.1.0.linux-arm64.tgz
	hashBin, err := hex.DecodeString(nfoA[0])
	if err != nil {
		return nil, err
	}

	res := &CockroachVersion{
		Filename: nfoA[1],
		hash:     hashBin,
	}

	return res, nil
}

func readStream(r io.Reader, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}
