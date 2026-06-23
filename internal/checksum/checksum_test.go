package checksum_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronelliott/gh-runx/internal/checksum"
)

// wrongSum is an arbitrary digest that matches no test file.
const wrongSum = "e6e3a1f3e2c0a3e1b9c2d6b8a5f4d3c2b1a0f9e8d7c6b5a4938271605f4e3d2c"

// TestIsManifestRecognizesCommonNames verifies that common sums-manifest names
// are recognized and unrelated assets are not.
func TestIsManifestRecognizesCommonNames(t *testing.T) {
	require.True(t, checksum.IsManifest("SHA256SUMS"))
	require.True(t, checksum.IsManifest("sha256sums.txt"))
	require.True(t, checksum.IsManifest("checksums.txt"))
	require.True(t, checksum.IsManifest("tool_1.0_checksums.txt"))
	require.False(t, checksum.IsManifest("tool-linux-amd64.tar.gz"))
	require.False(t, checksum.IsManifest("README.md"))
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

// TestLookupMissingAsset verifies that Lookup reports when the asset is absent.
func TestLookupMissingAsset(t *testing.T) {
	_, ok := checksum.Lookup("aaaa  other.tar.gz\n", "tool.tar.gz")
	require.False(t, ok)
}

// TestVerifyMatches verifies that Verify accepts a file whose digest matches.
func TestVerifyMatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))

	// sha256("hello")
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	require.NoError(t, checksum.Verify(path, want))
}

// TestVerifyMismatchReturnsErrMismatch verifies that Verify rejects a file whose
// digest differs and reports ErrMismatch.
func TestVerifyMismatchReturnsErrMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))

	err := checksum.Verify(path, wrongSum)
	require.ErrorIs(t, err, checksum.ErrMismatch)
}
