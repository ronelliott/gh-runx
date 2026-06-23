// Package archive unpacks release assets and selects the executable to run.
package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// archiveSuffixes are the supported archive extensions.
var archiveSuffixes = []string{".tar.gz", ".tgz", ".zip"}

// Prepare makes the asset at assetPath runnable and returns the path of the
// executable to run. Archives are unpacked into destDir; repoName selects the
// executable inside an archive and execOverride forces a specific relative path.
// When force is set, an already-unpacked destDir is discarded and re-extracted.
func Prepare(assetPath, destDir, repoName, execOverride string, force bool) (string, error) {
	if !isArchive(assetPath) {
		if err := os.Chmod(assetPath, 0o755); err != nil {
			return "", fmt.Errorf("archive: making %s executable: %w", assetPath, err)
		}
		return assetPath, nil
	}

	if err := unpack(assetPath, destDir, force); err != nil {
		return "", err
	}

	if execOverride != "" {
		return resolveOverride(destDir, execOverride)
	}
	return pickExecutable(destDir, repoName)
}

// isArchive reports whether name has a supported archive extension.
func isArchive(name string) bool {
	lower := strings.ToLower(name)
	for _, suf := range archiveSuffixes {
		if strings.HasSuffix(lower, suf) {
			return true
		}
	}
	return false
}

// pickExecutable selects the executable to run from an unpacked directory,
// preferring a file named repoName.
func pickExecutable(dir, repoName string) (string, error) {
	var executables []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0o111 != 0 {
			executables = append(executables, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("archive: scanning %s: %w", dir, err)
	}

	if len(executables) == 0 {
		return "", ErrNoExecutable
	}
	for _, path := range executables {
		if filepath.Base(path) == repoName {
			return path, nil
		}
	}
	if len(executables) == 1 {
		return executables[0], nil
	}
	return "", fmt.Errorf("%w: %s", ErrAmbiguousExecutable, strings.Join(baseNames(executables), ", "))
}

// resolveOverride validates and prepares an explicit executable path within destDir.
func resolveOverride(destDir, execOverride string) (string, error) {
	path, err := safeJoin(destDir, execOverride)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("archive: --exec %s: %w", execOverride, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("archive: --exec %s is a directory", execOverride)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		return "", fmt.Errorf("archive: making %s executable: %w", path, err)
	}
	return path, nil
}

// unpack extracts assetPath into destDir, skipping work if already unpacked.
// When force is set, an existing destDir is removed first so extraction is fresh.
func unpack(assetPath, destDir string, force bool) error {
	if force {
		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("archive: clearing %s: %w", destDir, err)
		}
	} else if entries, err := os.ReadDir(destDir); err == nil && len(entries) > 0 {
		return nil
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("archive: creating %s: %w", destDir, err)
	}

	lower := strings.ToLower(assetPath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return unpackZip(assetPath, destDir)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return unpackTarGz(assetPath, destDir)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedArchive, filepath.Base(assetPath))
	}
}

// unpackTarGz extracts a gzip-compressed tar archive into destDir.
func unpackTarGz(src, destDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("archive: opening %s: %w", src, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("archive: reading gzip %s: %w", src, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("archive: reading tar %s: %w", src, err)
		}

		target, err := safeJoin(destDir, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("archive: creating %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := writeFile(target, tr, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}
}

// unpackZip extracts a zip archive into destDir.
func unpackZip(src, destDir string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("archive: opening zip %s: %w", src, err)
	}
	defer r.Close()

	for _, file := range r.File {
		target, err := safeJoin(destDir, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("archive: creating %s: %w", target, err)
			}
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("archive: opening %s in zip: %w", file.Name, err)
		}
		if err := writeFile(target, rc, file.Mode()); err != nil {
			rc.Close()
			return err
		}
		rc.Close()
	}
	return nil
}

// baseNames returns the base name of each path.
func baseNames(paths []string) []string {
	names := make([]string, len(paths))
	for i, p := range paths {
		names[i] = filepath.Base(p)
	}
	return names
}

// safeJoin joins name onto destDir and rejects paths escaping destDir.
func safeJoin(destDir, name string) (string, error) {
	target := filepath.Join(destDir, name)
	cleaned := filepath.Clean(target)
	if cleaned != destDir && !strings.HasPrefix(cleaned, destDir+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive: unsafe path %q in archive", name)
	}
	return target, nil
}

// writeFile creates target with mode and copies r into it.
func writeFile(target string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("archive: creating %s: %w", filepath.Dir(target), err)
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode|0o200)
	if err != nil {
		return fmt.Errorf("archive: creating %s: %w", target, err)
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return fmt.Errorf("archive: writing %s: %w", target, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("archive: closing %s: %w", target, err)
	}
	return nil
}
