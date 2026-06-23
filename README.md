# gh-runx

[![CI](https://github.com/ronelliott/gh-runx/actions/workflows/ci.yml/badge.svg)](https://github.com/ronelliott/gh-runx/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE.md)

Run any GitHub release straight from the command line. `gh-runx` is a
[GitHub CLI](https://cli.github.com) extension that fetches the release asset for
your current platform, caches it, and runs it — in one step.

```console
$ gh runx ronelliott/muzak
# → downloads muzak-darwin-arm64.tar.gz, unpacks it, runs muzak
```

No manual "find the right asset, download, untar, chmod +x, run" dance. It works
with public and private repositories, because it authenticates with your
existing `gh` login.

## Features

- **One command** to download and run a release binary for the right OS/arch.
- **Private repositories** work out of the box — assets are fetched through the
  authenticated GitHub API.
- **Smart asset matching** by OS and architecture, with common naming aliases
  (`darwin`/`macos`, `amd64`/`x86_64`, `arm64`/`aarch64`, …).
- **Archive aware** — transparently unpacks `.tar.gz`, `.tgz`, and `.zip`, or
  runs a bare binary as-is.
- **Cached** — a release is downloaded once and reused on later runs.
- **Argument forwarding** — everything after `--` is passed straight to the
  program.

## Installation

```sh
gh extension install ronelliott/gh-runx
```

Upgrade later with `gh extension upgrade runx`.

## Usage

```
gh runx <org/repo> [--version tag] [--exec path] [--force] [-- args...]
```

| Flag | Description |
| --- | --- |
| `--version <tag>` | Release tag to run. Defaults to the latest release. |
| `--exec <path>` | Relative path of the executable to run inside an archive. |
| `--force`, `-f` | Re-download and re-unpack even if a cached copy exists. |
| `-- <args...>` | Everything after `--` is forwarded to the program. |

Flags may appear before or after the `org/repo` argument.

### Examples

```sh
# Run the latest release
gh runx ronelliott/muzak

# Run a specific release
gh runx ronelliott/muzak --version v0.1.10

# Forward arguments to the program
gh runx ronelliott/muzak -- --help

# Pick a specific binary inside an archive that ships several
gh runx some/tool --exec bin/tool

# Force a fresh download, ignoring the cache
gh runx ronelliott/muzak --force
```

## How it works

1. **Resolve the release.** Looks up the latest release (or `--version <tag>`)
   for `org/repo` via the GitHub API, authenticated with your `gh` token.
2. **Match the asset.** Selects the single release asset whose name matches the
   current `GOOS`/`GOARCH`, using common aliases. Checksums, signatures, and
   packages (`.sha256`, `.sig`, `.deb`, `.rpm`, `.pkg`, `.msi`, …) are ignored.
3. **Download and cache.** Fetches the asset through the GitHub asset API
   (so private releases work) into the cache directory, reusing it next time.
4. **Unpack.** Extracts `.tar.gz`/`.tgz`/`.zip` archives; a bare binary is used
   directly.
5. **Pick the executable.** Prefers a file named after the repository, falls
   back to the only executable present, or asks you to disambiguate with
   `--exec`.
6. **Run.** Replaces the current process with the program (a child process on
   Windows), forwarding your arguments and environment.

## Caching

Downloads are cached under your user config directory:

| OS | Location |
| --- | --- |
| macOS | `~/Library/Application Support/gh-runx/cache` |
| Linux | `$XDG_CONFIG_HOME/gh-runx/cache` (or `~/.config/gh-runx/cache`) |
| Windows | `%AppData%\gh-runx\cache` |

The layout is `<owner>/<repo>/<tag>/`, holding the downloaded asset and an
`unpacked/` tree. Use `--force` to refresh a cached release, or delete a tag
directory to clear it.

## Requirements

- [GitHub CLI](https://cli.github.com) (`gh`), authenticated (`gh auth login`).
- [Go](https://go.dev) 1.26+ — only to build from source.

## Building from source

```sh
git clone https://github.com/ronelliott/gh-runx
cd gh-runx
go build -o gh-runx .
gh extension install .
```

## License

[MIT](LICENSE.md) © Ron Elliott
