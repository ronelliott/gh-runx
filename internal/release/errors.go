package release

import "errors"

// ErrAmbiguousAsset indicates more than one release asset matches the platform.
var ErrAmbiguousAsset = errors.New("multiple release assets match the platform")

// ErrNoMatchingAsset indicates no release asset matches the platform.
var ErrNoMatchingAsset = errors.New("no release asset matches the platform")

// ErrReleaseNotFound indicates the requested release does not exist.
var ErrReleaseNotFound = errors.New("release not found")
