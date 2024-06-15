package froach

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/KarpelesLab/fileutil"
)

type CockroachVersion struct {
	Filename string
	hash     []byte
}

// Exe returns the path to cockroach latest version
func Exe() (string, error) {
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

	resp, err := http.Get("https://binaries.cockroachdb.com/" + v.Filename)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP status: %s", resp.Status)
	}

	h := sha256.New()
	r := io.TeeReader(resp.Body, h)

	_, err = io.Copy(fp, r)
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
	resp, err := http.Get("https://binaries.cockroachdb.com/" + v.Filename)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP status: %s", resp.Status)
	}

	h := sha256.New()
	r := io.TeeReader(resp.Body, h)
	g, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	err = fileutil.TarExtract(g, dirname)
	if err != nil {
		return err
	}

	return nil
}

func GetLatestVersion() (*CockroachVersion, error) {
	return GetVersion("latest")
}

func GetVersion(vers string) (*CockroachVersion, error) {
	// https://binaries.cockroachdb.com/cockroach-$vers.linux-amd64.tgz.sha256sum
	u := fmt.Sprintf("https://binaries.cockroachdb.com/cockroach-%s.%s-%s.tgz.sha256sum", vers, runtime.GOOS, runtime.GOARCH)
	nfo, err := simpleGet(u)
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

func simpleGet(u string) ([]byte, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}