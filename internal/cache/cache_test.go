package cache_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronelliott/gh-runx/internal/cache"
)

// stubDoer is a doer that records calls and returns a fixed body.
type stubDoer struct {
	body  string
	calls int
}

// Request returns a 200 response with the stub body and counts the call.
func (s *stubDoer) Request(method, url string, body io.Reader) (*http.Response, error) {
	s.calls++
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(s.body)),
	}, nil
}

// TestEnsureAssetDownloadsThenCaches verifies that EnsureAsset downloads on the
// first call and reuses the cached file on the second.
func TestEnsureAssetDownloadsThenCaches(t *testing.T) {
	root := t.TempDir()
	doer := &stubDoer{body: "binary-bytes"}
	store, err := cache.New(root, doer)
	require.NoError(t, err)

	path, downloaded, err := store.EnsureAsset("octocat", "tool", "v1", "tool.tar.gz", "https://example.com/a", false)
	require.NoError(t, err)
	require.True(t, downloaded)
	require.Equal(t, filepath.Join(root, "octocat", "tool", "v1", "tool.tar.gz"), path)
	require.Equal(t, 1, doer.calls)

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "binary-bytes", string(contents))

	again, downloaded, err := store.EnsureAsset("octocat", "tool", "v1", "tool.tar.gz", "https://example.com/a", false)
	require.NoError(t, err)
	require.False(t, downloaded)
	require.Equal(t, path, again)
	require.Equal(t, 1, doer.calls)
}

// TestEnsureAssetForceRedownloadsCachedFile verifies that force re-downloads an
// asset that is already cached.
func TestEnsureAssetForceRedownloadsCachedFile(t *testing.T) {
	root := t.TempDir()
	doer := &stubDoer{body: "binary-bytes"}
	store, err := cache.New(root, doer)
	require.NoError(t, err)

	path, _, err := store.EnsureAsset("octocat", "tool", "v1", "tool.tar.gz", "https://example.com/a", false)
	require.NoError(t, err)
	require.Equal(t, 1, doer.calls)

	again, downloaded, err := store.EnsureAsset("octocat", "tool", "v1", "tool.tar.gz", "https://example.com/a", true)
	require.NoError(t, err)
	require.True(t, downloaded)
	require.Equal(t, path, again)
	require.Equal(t, 2, doer.calls)
}

// TestEnsureAssetReportsBadStatus verifies that a non-200 response surfaces an
// error and writes no file.
func TestEnsureAssetReportsBadStatus(t *testing.T) {
	root := t.TempDir()
	doer := &errorDoer{}
	store, err := cache.New(root, doer)
	require.NoError(t, err)

	_, _, err = store.EnsureAsset("octocat", "tool", "v1", "tool.tar.gz", "https://example.com/a", false)
	require.Error(t, err)

	_, statErr := os.Stat(filepath.Join(root, "octocat", "tool", "v1", "tool.tar.gz"))
	require.True(t, os.IsNotExist(statErr))
}

// TestEnsureAssetRejectsUnsafeAssetName verifies that an asset name containing
// path separators is rejected and no file is written outside the cache.
func TestEnsureAssetRejectsUnsafeAssetName(t *testing.T) {
	root := t.TempDir()
	doer := &stubDoer{body: "binary-bytes"}
	store, err := cache.New(root, doer)
	require.NoError(t, err)

	_, _, err = store.EnsureAsset("octocat", "tool", "v1", "../../evil", "https://example.com/a", false)
	require.Error(t, err)
	require.Equal(t, 0, doer.calls)
}

// errorDoer is a doer that returns a 404 response.
type errorDoer struct{}

// Request returns a 404 response with an empty body.
func (errorDoer) Request(method, url string, body io.Reader) (*http.Response, error) {
	return &http.Response{
		Status:     "404 Not Found",
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

// TestNewRejectsEmptyRoot verifies that New rejects an empty root directory.
func TestNewRejectsEmptyRoot(t *testing.T) {
	store, err := cache.New("", &stubDoer{})
	require.Error(t, err)
	require.Nil(t, store)
}
