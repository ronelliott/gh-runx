package archive

import "errors"

// ErrAmbiguousExecutable indicates several executables were found and none could
// be chosen unambiguously.
var ErrAmbiguousExecutable = errors.New("multiple executables found; choose one with --exec")

// ErrNoExecutable indicates no runnable executable was found in the asset.
var ErrNoExecutable = errors.New("no executable found in release asset")

// ErrUnsupportedArchive indicates the asset is not a supported archive format.
var ErrUnsupportedArchive = errors.New("unsupported archive format")
