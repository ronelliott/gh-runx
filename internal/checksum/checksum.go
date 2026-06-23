// Package checksum verifies a downloaded asset against a SHA-256 sums manifest
// published alongside it in a release.
package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ronelliott/gh-runx/internal/release"
)

// ErrMismatch indicates a file's SHA-256 digest does not match the manifest.
var ErrMismatch = errors.New("checksum mismatch")

// signatureSuffixes are detached-signature extensions that wrap a manifest but
// are not themselves a sums manifest.
var signatureSuffixes = []string{".asc", ".pem", ".sig"}

// Outcome reports how an asset was treated by VerifyAsset.
type Outcome int

const (
	// NoManifest means the release published no sums manifest to verify against.
	NoManifest Outcome = iota

	// Unlisted means a manifest exists but does not list the asset.
	Unlisted

	// Verified means the asset's digest matched the manifest.
	Verified
)

// requester fetches a URL through the authenticated go-gh REST client.
type requester interface {
	Request(method, path string, body io.Reader) (*http.Response, error)
}

// VerifyAsset verifies target's downloaded file at path against a SHA-256 sums
// manifest among assets, fetching the manifest through fetch. On a digest
// mismatch or a manifest-fetch failure it removes the file at path so a later
// run re-downloads rather than trusting an unverified copy, and returns an
// error. It returns NoManifest or Unlisted (with a nil error) when there is
// nothing to verify against.
func VerifyAsset(fetch requester, assets []release.Asset, target release.Asset, path string) (Outcome, error) {
	manifest, ok := findManifest(assets)
	if !ok {
		return NoManifest, nil
	}

	body, err := fetchBytes(fetch, manifest.URL)
	if err != nil {
		os.Remove(path)
		return NoManifest, fmt.Errorf("checksum: fetching %s: %w", manifest.Name, err)
	}

	want, ok := Lookup(string(body), target.Name)
	if !ok {
		return Unlisted, nil
	}

	if err := Verify(path, want); err != nil {
		os.Remove(path)
		return NoManifest, err
	}
	return Verified, nil
}

// IsManifest reports whether name looks like a SHA-256 sums manifest covering a
// release's assets (for example "SHA256SUMS" or "checksums.txt"). Detached
// signatures of a manifest are not treated as a manifest.
func IsManifest(name string) bool {
	lower := strings.ToLower(name)
	for _, suf := range signatureSuffixes {
		if strings.HasSuffix(lower, suf) {
			return false
		}
	}
	return lower == "sha256sums" ||
		lower == "sha256sums.txt" ||
		strings.Contains(lower, "checksums")
}

// Lookup returns the hex SHA-256 digest recorded for assetName in a sums
// manifest. Manifest lines are "<hex-digest><space><indicator><name>", where
// the indicator is a space (text mode) or "*" (binary mode); the name may
// itself contain spaces.
func Lookup(manifest, assetName string) (string, bool) {
	for line := range strings.SplitSeq(manifest, "\n") {
		line = strings.TrimRight(line, "\r")
		i := strings.IndexByte(line, ' ')
		if i <= 0 {
			continue
		}
		digest := line[:i]
		name := line[i+1:]
		if len(name) > 0 && (name[0] == ' ' || name[0] == '*') {
			name = name[1:]
		}
		if name == assetName {
			return digest, true
		}
	}
	return "", false
}

// Verify checks that the file at path has the expected hex SHA-256 digest.
func Verify(path, wantHex string) error {
	got, err := sumFile(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, wantHex) {
		return fmt.Errorf("%w: %s has %s, expected %s", ErrMismatch, filepath.Base(path), got, wantHex)
	}
	return nil
}

// findManifest returns the first asset that looks like a sums manifest.
func findManifest(assets []release.Asset) (release.Asset, bool) {
	for _, a := range assets {
		if IsManifest(a.Name) {
			return a, true
		}
	}
	return release.Asset{}, false
}

// fetchBytes downloads the bytes at url through the authenticated client.
func fetchBytes(fetch requester, url string) ([]byte, error) {
	resp, err := fetch.Request(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// sumFile returns the hex SHA-256 digest of the file at path.
func sumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("checksum: opening %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("checksum: hashing %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
