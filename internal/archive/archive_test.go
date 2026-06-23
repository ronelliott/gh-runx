package archive_test

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronelliott/gh-runx/internal/archive"
)

// tarEntry describes a file to place in a test tar.gz archive.
type tarEntry struct {
	name string

	body string

	mode int64
}

// writeTarGz creates a gzip-compressed tar archive containing entries.
func writeTarGz(t *testing.T, path string, entries []tarEntry) {
	t.Helper()

	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	for _, e := range entries {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name:     e.name,
			Mode:     e.mode,
			Size:     int64(len(e.body)),
			Typeflag: tar.TypeReg,
		}))
		_, err := tw.Write([]byte(e.body))
		require.NoError(t, err)
	}
}

// TestPrepareUnpacksAndPicksRepoNamedExecutable verifies that Prepare unpacks an
// archive and selects the executable whose name matches the repository.
func TestPrepareUnpacksAndPicksRepoNamedExecutable(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "tool.tar.gz")
	writeTarGz(t, asset, []tarEntry{
		{name: "README.md", body: "docs", mode: 0o644},
		{name: "tool", body: "#!/bin/sh\n", mode: 0o755},
		{name: "helper", body: "#!/bin/sh\n", mode: 0o755},
	})

	dest := filepath.Join(dir, "unpacked")
	got, err := archive.Prepare(asset, dest, "tool", "", false)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dest, "tool"), got)
}

// TestPrepareRunsSingleExecutable verifies that Prepare runs the only executable
// when none matches the repository name.
func TestPrepareRunsSingleExecutable(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "tool.tar.gz")
	writeTarGz(t, asset, []tarEntry{
		{name: "README.md", body: "docs", mode: 0o644},
		{name: "renamed-binary", body: "#!/bin/sh\n", mode: 0o755},
	})

	dest := filepath.Join(dir, "unpacked")
	got, err := archive.Prepare(asset, dest, "tool", "", false)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dest, "renamed-binary"), got)
}

// TestPrepareAmbiguousExecutablesReturnsError verifies that Prepare reports an
// error when several executables exist and none matches the repository name.
func TestPrepareAmbiguousExecutablesReturnsError(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "tool.tar.gz")
	writeTarGz(t, asset, []tarEntry{
		{name: "one", body: "#!/bin/sh\n", mode: 0o755},
		{name: "two", body: "#!/bin/sh\n", mode: 0o755},
	})

	dest := filepath.Join(dir, "unpacked")
	_, err := archive.Prepare(asset, dest, "tool", "", false)
	require.ErrorIs(t, err, archive.ErrAmbiguousExecutable)
}

// TestPrepareNoExecutableReturnsError verifies that Prepare reports an error
// when the archive contains no executable file.
func TestPrepareNoExecutableReturnsError(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "tool.tar.gz")
	writeTarGz(t, asset, []tarEntry{
		{name: "README.md", body: "docs", mode: 0o644},
	})

	dest := filepath.Join(dir, "unpacked")
	_, err := archive.Prepare(asset, dest, "tool", "", false)
	require.ErrorIs(t, err, archive.ErrNoExecutable)
}

// TestPrepareExecOverrideSelectsPath verifies that Prepare honors an explicit
// executable path within the archive.
func TestPrepareExecOverrideSelectsPath(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "tool.tar.gz")
	writeTarGz(t, asset, []tarEntry{
		{name: "bin/runme", body: "#!/bin/sh\n", mode: 0o644},
		{name: "tool", body: "#!/bin/sh\n", mode: 0o755},
	})

	dest := filepath.Join(dir, "unpacked")
	got, err := archive.Prepare(asset, dest, "tool", "bin/runme", false)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dest, "bin", "runme"), got)

	info, err := os.Stat(got)
	require.NoError(t, err)
	require.NotZero(t, info.Mode().Perm()&0o100)
}

// TestPrepareBareBinaryIsMadeExecutable verifies that a non-archive asset is
// made executable and returned as-is.
func TestPrepareBareBinaryIsMadeExecutable(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "tool-darwin-arm64")
	require.NoError(t, os.WriteFile(asset, []byte("#!/bin/sh\n"), 0o644))

	got, err := archive.Prepare(asset, filepath.Join(dir, "unpacked"), "tool", "", false)
	require.NoError(t, err)
	require.Equal(t, asset, got)

	info, err := os.Stat(got)
	require.NoError(t, err)
	require.NotZero(t, info.Mode().Perm()&0o100)
}

// TestPrepareFailedUnpackLeavesNoDestination verifies that a failed extraction
// leaves no destination directory behind, so it is not mistaken for a complete
// unpack on a later run.
func TestPrepareFailedUnpackLeavesNoDestination(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "corrupt.tar.gz")
	require.NoError(t, os.WriteFile(asset, []byte("not a gzip stream"), 0o644))

	dest := filepath.Join(dir, "unpacked")
	_, err := archive.Prepare(asset, dest, "tool", "", false)
	require.Error(t, err)

	_, statErr := os.Stat(dest)
	require.True(t, os.IsNotExist(statErr))
}

// TestPrepareForceReunpacksCleansDestination verifies that force re-extracts the
// archive into a fresh destination, discarding stale unpacked files.
func TestPrepareForceReunpacksCleansDestination(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "tool.tar.gz")
	writeTarGz(t, asset, []tarEntry{
		{name: "tool", body: "#!/bin/sh\n", mode: 0o755},
	})

	dest := filepath.Join(dir, "unpacked")
	_, err := archive.Prepare(asset, dest, "tool", "", false)
	require.NoError(t, err)

	// A stale file from a prior unpack must not survive a forced re-extract.
	stale := filepath.Join(dest, "stale")
	require.NoError(t, os.WriteFile(stale, []byte("old"), 0o644))

	got, err := archive.Prepare(asset, dest, "tool", "", true)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dest, "tool"), got)

	_, statErr := os.Stat(stale)
	require.True(t, os.IsNotExist(statErr))
}
