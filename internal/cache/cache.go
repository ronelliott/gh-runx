// Package cache stores downloaded release assets under a root directory and
// downloads them on demand.
package cache

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// doer performs authenticated HTTP requests for asset downloads. It matches
// the go-gh REST client's Request method, which carries the GitHub token and
// follows the asset API redirect so private-repo assets download.
type doer interface {
	Request(method, path string, body io.Reader) (*http.Response, error)
}

// Store manages downloaded release assets under a root directory.
type Store struct {
	client doer

	root string
}

// New constructs a Store rooted at root using client for downloads.
func New(root string, client doer) (*Store, error) {
	if root == "" {
		return nil, errors.New("cache: empty root")
	}
	if client == nil {
		return nil, errors.New("cache: nil HTTP client")
	}
	return &Store{client: client, root: root}, nil
}

// DefaultRoot returns the default cache root under the user config directory.
func DefaultRoot() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cache: locating user config dir: %w", err)
	}
	return filepath.Join(dir, "gh-runx", "cache"), nil
}

// EnsureAsset downloads asset to the tag directory and returns its path along
// with whether a download occurred. A cached file is reused (downloaded=false)
// unless force is set, which always re-downloads.
func (s *Store) EnsureAsset(owner, repo, tag, assetName, url string, force bool) (path string, downloaded bool, err error) {
	if !safeName(assetName) {
		return "", false, fmt.Errorf("cache: unsafe asset name %q", assetName)
	}

	dir := s.TagDir(owner, repo, tag)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false, fmt.Errorf("cache: creating %s: %w", dir, err)
	}

	dest := filepath.Join(dir, assetName)
	if !force {
		if _, err := os.Stat(dest); err == nil {
			return dest, false, nil
		}
	}

	if err := s.download(url, dest); err != nil {
		return "", false, err
	}
	return dest, true, nil
}

// safeName reports whether assetName is a plain file name that cannot escape
// the cache directory. The name comes from publisher-controlled release JSON.
func safeName(assetName string) bool {
	if assetName == "" || assetName == "." || assetName == ".." {
		return false
	}
	return assetName == filepath.Base(assetName)
}

// TagDir returns the directory holding artifacts for owner/repo at tag.
func (s *Store) TagDir(owner, repo, tag string) string {
	return filepath.Join(s.root, owner, repo, tag)
}

// download fetches url and writes it atomically to dest.
func (s *Store) download(url, dest string) error {
	resp, err := s.client.Request(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("cache: downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cache: downloading %s: unexpected status %s", url, resp.Status)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".download-*")
	if err != nil {
		return fmt.Errorf("cache: creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("cache: writing download: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("cache: closing download: %w", err)
	}

	if err := os.Rename(tmpName, dest); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("cache: finalizing download: %w", err)
	}
	return nil
}
