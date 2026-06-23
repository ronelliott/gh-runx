package release_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronelliott/gh-runx/internal/release"
)

// TestMatchAssetSelectsPlatformAsset verifies that MatchAsset returns the asset
// whose name contains the platform OS and architecture.
func TestMatchAssetSelectsPlatformAsset(t *testing.T) {
	assets := []release.Asset{
		{Name: "tool_linux_amd64.tar.gz"},
		{Name: "tool_darwin_arm64.tar.gz"},
		{Name: "tool_windows_amd64.zip"},
	}

	got, err := release.MatchAsset(assets, "darwin", "arm64")
	require.NoError(t, err)
	require.Equal(t, "tool_darwin_arm64.tar.gz", got.Name)
}

// TestMatchAssetMatchesAliases verifies that MatchAsset matches common naming
// aliases such as macos and x86_64.
func TestMatchAssetMatchesAliases(t *testing.T) {
	assets := []release.Asset{
		{Name: "tool-macos-x86_64.zip"},
		{Name: "tool-linux-aarch64.tar.gz"},
	}

	got, err := release.MatchAsset(assets, "darwin", "amd64")
	require.NoError(t, err)
	require.Equal(t, "tool-macos-x86_64.zip", got.Name)
}

// TestMatchAssetNoMatchReturnsError verifies that MatchAsset returns
// ErrNoMatchingAsset when nothing matches the platform.
func TestMatchAssetNoMatchReturnsError(t *testing.T) {
	assets := []release.Asset{{Name: "tool_windows_amd64.zip"}}

	got, err := release.MatchAsset(assets, "linux", "arm64")
	require.ErrorIs(t, err, release.ErrNoMatchingAsset)
	require.Equal(t, release.Asset{}, got)
}

// TestMatchAssetAmbiguousReturnsError verifies that MatchAsset returns
// ErrAmbiguousAsset when several assets match the platform.
func TestMatchAssetAmbiguousReturnsError(t *testing.T) {
	assets := []release.Asset{
		{Name: "tool_darwin_arm64.tar.gz"},
		{Name: "tool_darwin_arm64_extra.zip"},
	}

	got, err := release.MatchAsset(assets, "darwin", "arm64")
	require.ErrorIs(t, err, release.ErrAmbiguousAsset)
	require.Equal(t, release.Asset{}, got)
}

// TestMatchAssetSkipsNonRunnableAssets verifies that MatchAsset ignores
// checksums and signatures even when they match the platform name.
func TestMatchAssetSkipsNonRunnableAssets(t *testing.T) {
	assets := []release.Asset{
		{Name: "tool_darwin_arm64.tar.gz"},
		{Name: "tool_darwin_arm64.tar.gz.sha256"},
		{Name: "tool_darwin_arm64.sig"},
	}

	got, err := release.MatchAsset(assets, "darwin", "arm64")
	require.NoError(t, err)
	require.Equal(t, "tool_darwin_arm64.tar.gz", got.Name)
}
