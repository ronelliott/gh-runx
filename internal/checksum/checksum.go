// Package checksum verifies a downloaded asset against a SHA-256 sums manifest
// published alongside it in a release.
package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrMismatch indicates a file's SHA-256 digest does not match the manifest.
var ErrMismatch = errors.New("checksum mismatch")

// IsManifest reports whether name looks like a SHA-256 sums manifest covering a
// release's assets (for example "SHA256SUMS" or "checksums.txt").
func IsManifest(name string) bool {
	lower := strings.ToLower(name)
	return lower == "sha256sums" ||
		lower == "sha256sums.txt" ||
		strings.Contains(lower, "checksums")
}

// Lookup returns the hex SHA-256 digest recorded for assetName in a sums
// manifest. Manifest lines are "<hex-digest>  <name>", optionally with a "*"
// before the name to mark binary mode.
func Lookup(manifest, assetName string) (string, bool) {
	for line := range strings.SplitSeq(manifest, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if name == assetName {
			return fields[0], true
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
		return fmt.Errorf("%w: %s has %s, expected %s", ErrMismatch, path, got, wantHex)
	}
	return nil
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
