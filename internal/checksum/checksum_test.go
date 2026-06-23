package checksum_test

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronelliott/gh-runx/internal/checksum"
	"github.com/ronelliott/gh-runx/internal/release"
)

// sha256 of "hello".
const helloSum = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

// wrongSum is an arbitrary digest that matches no test file.
const wrongSum = "0000000000000000000000000000000000000000000000000000000000000000"

// stubRequester returns a canned response or error for any request.
type stubRequester struct {
	body   string
	err    error
	status int
}

// Request returns the stub response, defaulting to status 200.
func (s stubRequester) Request(method, url string, body io.Reader) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}
	status := s.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(s.body)),
	}, nil
}

// writeHello writes a file containing "hello" and returns its path.
func writeHello(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "asset.tar.gz")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))
	return path
}

// assetsWithManifest returns a manifest asset plus the target asset.
func assetsWithManifest() ([]release.Asset, release.Asset) {
	target := release.Asset{Name: "asset.tar.gz", URL: "https://example.com/a"}
	return []release.Asset{
		{Name: "SHA256SUMS", URL: "https://example.com/sums"},
		target,
	}, target
}

// TestIsManifestRecognizesCommonNames verifies that common sums-manifest names
// are recognized and unrelated assets and signatures are not.
func TestIsManifestRecognizesCommonNames(t *testing.T) {
	require.True(t, checksum.IsManifest("SHA256SUMS"))
	require.True(t, checksum.IsManifest("sha256sums.txt"))
	require.True(t, checksum.IsManifest("checksums.txt"))
	require.True(t, checksum.IsManifest("tool_1.0_checksums.txt"))
	require.False(t, checksum.IsManifest("tool-linux-amd64.tar.gz"))
	require.False(t, checksum.IsManifest("README.md"))
	require.False(t, checksum.IsManifest("SHA256SUMS.sig"))
	require.False(t, checksum.IsManifest("checksums.txt.asc"))
}

// TestLookupFindsDigest verifies that Lookup returns the digest for a named
// asset across plain and binary-mode ("*name") manifest lines.
func TestLookupFindsDigest(t *testing.T) {
	manifest := "aaaa  tool-linux-amd64.tar.gz\nbbbb *tool-darwin-arm64.tar.gz\n"

	got, ok := checksum.Lookup(manifest, "tool-linux-amd64.tar.gz")
	require.True(t, ok)
	require.Equal(t, "aaaa", got)

	got, ok = checksum.Lookup(manifest, "tool-darwin-arm64.tar.gz")
	require.True(t, ok)
	require.Equal(t, "bbbb", got)
}

// TestLookupHandlesSpacesInName verifies that asset names containing spaces are
// parsed correctly.
func TestLookupHandlesSpacesInName(t *testing.T) {
	got, ok := checksum.Lookup("aaaa  my tool v2.tar.gz\n", "my tool v2.tar.gz")
	require.True(t, ok)
	require.Equal(t, "aaaa", got)
}

// TestLookupHandlesCarriageReturns verifies that CRLF manifests are parsed.
func TestLookupHandlesCarriageReturns(t *testing.T) {
	got, ok := checksum.Lookup("aaaa  tool.tar.gz\r\n", "tool.tar.gz")
	require.True(t, ok)
	require.Equal(t, "aaaa", got)
}

// TestLookupMissingAsset verifies that Lookup reports when the asset is absent,
// including for an empty manifest.
func TestLookupMissingAsset(t *testing.T) {
	_, ok := checksum.Lookup("aaaa  other.tar.gz\n", "tool.tar.gz")
	require.False(t, ok)

	_, ok = checksum.Lookup("", "tool.tar.gz")
	require.False(t, ok)
}

// TestVerifyMatches verifies that Verify accepts a file whose digest matches.
func TestVerifyMatches(t *testing.T) {
	require.NoError(t, checksum.Verify(writeHello(t), helloSum))
}

// TestVerifyMismatchReturnsErrMismatch verifies that Verify rejects a file whose
// digest differs and reports ErrMismatch.
func TestVerifyMismatchReturnsErrMismatch(t *testing.T) {
	err := checksum.Verify(writeHello(t), wrongSum)
	require.ErrorIs(t, err, checksum.ErrMismatch)
}

// TestVerifyAssetVerifiesListedAsset verifies that a listed, matching asset is
// reported Verified and the file is kept.
func TestVerifyAssetVerifiesListedAsset(t *testing.T) {
	path := writeHello(t)
	assets, target := assetsWithManifest()
	fetch := stubRequester{body: helloSum + "  asset.tar.gz\n"}

	outcome, err := checksum.VerifyAsset(fetch, assets, target, path)
	require.NoError(t, err)
	require.Equal(t, checksum.Verified, outcome)
	require.FileExists(t, path)
}

// TestVerifyAssetNoManifest verifies that a release without a manifest returns
// NoManifest and keeps the file.
func TestVerifyAssetNoManifest(t *testing.T) {
	path := writeHello(t)
	target := release.Asset{Name: "asset.tar.gz", URL: "https://example.com/a"}
	assets := []release.Asset{target}

	outcome, err := checksum.VerifyAsset(stubRequester{}, assets, target, path)
	require.NoError(t, err)
	require.Equal(t, checksum.NoManifest, outcome)
	require.FileExists(t, path)
}

// TestVerifyAssetUnlisted verifies that a manifest omitting the asset returns
// Unlisted and keeps the file.
func TestVerifyAssetUnlisted(t *testing.T) {
	path := writeHello(t)
	assets, target := assetsWithManifest()
	fetch := stubRequester{body: helloSum + "  other.tar.gz\n"}

	outcome, err := checksum.VerifyAsset(fetch, assets, target, path)
	require.NoError(t, err)
	require.Equal(t, checksum.Unlisted, outcome)
	require.FileExists(t, path)
}

// TestVerifyAssetMismatchRemovesFile verifies that a digest mismatch errors and
// removes the cached file.
func TestVerifyAssetMismatchRemovesFile(t *testing.T) {
	path := writeHello(t)
	assets, target := assetsWithManifest()
	fetch := stubRequester{body: wrongSum + "  asset.tar.gz\n"}

	_, err := checksum.VerifyAsset(fetch, assets, target, path)
	require.ErrorIs(t, err, checksum.ErrMismatch)
	require.NoFileExists(t, path)
}

// TestVerifyAssetManifestFetchFailureRemovesFile verifies that a manifest-fetch
// failure errors and removes the cached file, so verification cannot be bypassed
// by a later cache hit.
func TestVerifyAssetManifestFetchFailureRemovesFile(t *testing.T) {
	path := writeHello(t)
	assets, target := assetsWithManifest()
	fetch := stubRequester{err: errors.New("network down")}

	_, err := checksum.VerifyAsset(fetch, assets, target, path)
	require.Error(t, err)
	require.NoFileExists(t, path)
}

// TestVerifyAssetManifestBadStatusRemovesFile verifies that a non-200 manifest
// response errors and removes the cached file.
func TestVerifyAssetManifestBadStatusRemovesFile(t *testing.T) {
	path := writeHello(t)
	assets, target := assetsWithManifest()
	fetch := stubRequester{status: http.StatusNotFound}

	_, err := checksum.VerifyAsset(fetch, assets, target, path)
	require.Error(t, err)
	require.NoFileExists(t, path)
}
