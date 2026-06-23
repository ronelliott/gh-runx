package release

import (
	"fmt"
	"strings"
)

// archAliases maps a GOARCH value to substrings that identify it in asset names.
var archAliases = map[string][]string{
	"386":   {"386", "i386"},
	"amd64": {"amd64", "x86_64", "x64"},
	"arm64": {"arm64", "aarch64"},
}

// osAliases maps a GOOS value to substrings that identify it in asset names.
var osAliases = map[string][]string{
	"darwin":  {"darwin", "macos", "osx"},
	"linux":   {"linux"},
	"windows": {"windows"},
}

// skipSuffixes are asset suffixes that never contain a runnable program.
var skipSuffixes = []string{
	".asc", ".deb", ".msi", ".pem", ".pkg", ".rpm", ".sbom", ".sha256", ".sha512", ".sig", ".txt",
}

// MatchAsset returns the single asset matching goos and goarch. It returns
// ErrNoMatchingAsset when nothing matches and ErrAmbiguousAsset when several do.
func MatchAsset(assets []Asset, goos, goarch string) (Asset, error) {
	osNames := osAliases[goos]
	archNames := archAliases[goarch]

	var matches []Asset
	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if hasSkipSuffix(name) {
			continue
		}
		if containsAny(name, osNames) && containsAny(name, archNames) {
			matches = append(matches, a)
		}
	}

	switch len(matches) {
	case 0:
		return Asset{}, ErrNoMatchingAsset
	case 1:
		return matches[0], nil
	default:
		return Asset{}, fmt.Errorf("%w: %s", ErrAmbiguousAsset, strings.Join(assetNames(matches), ", "))
	}
}

// assetNames returns the names of the given assets.
func assetNames(assets []Asset) []string {
	names := make([]string, len(assets))
	for i, a := range assets {
		names[i] = a.Name
	}
	return names
}

// containsAny reports whether s contains any of subs.
func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// hasSkipSuffix reports whether name ends with a non-runnable suffix.
func hasSkipSuffix(name string) bool {
	for _, suf := range skipSuffixes {
		if strings.HasSuffix(name, suf) {
			return true
		}
	}
	return false
}
