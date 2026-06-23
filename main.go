package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/ronelliott/gh-runx/internal/archive"
	"github.com/ronelliott/gh-runx/internal/cache"
	"github.com/ronelliott/gh-runx/internal/release"
	"github.com/ronelliott/gh-runx/internal/runner"
)

// usage describes the command-line interface.
const usage = "usage: gh runx <org/repo> [--version tag] [--exec path] [-- args...]"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gh-runx:", err)
		os.Exit(1)
	}
}

// run parses arguments, resolves and runs the matching release asset.
func run(args []string) error {
	var forwardArgs []string
	if idx := indexOf(args, "--"); idx >= 0 {
		forwardArgs = args[idx+1:]
		args = args[:idx]
	}

	fs := flag.NewFlagSet("gh-runx", flag.ContinueOnError)
	version := fs.String("version", "", "release tag to run (default: latest)")
	execPath := fs.String("exec", "", "path within the archive to execute")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New(usage)
	}

	owner, repo, err := splitRepo(fs.Arg(0))
	if err != nil {
		return err
	}

	rest, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("creating REST client: %w", err)
	}
	releases, err := release.New(rest)
	if err != nil {
		return err
	}

	rel, err := resolveRelease(releases, owner, repo, *version)
	if err != nil {
		return err
	}

	asset, err := release.MatchAsset(rel.Assets, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return fmt.Errorf("%w (%s/%s, release %s)", err, runtime.GOOS, runtime.GOARCH, rel.Tag)
	}

	root, err := cache.DefaultRoot()
	if err != nil {
		return err
	}
	httpClient, err := api.DefaultHTTPClient()
	if err != nil {
		return fmt.Errorf("creating HTTP client: %w", err)
	}
	store, err := cache.New(root, httpClient)
	if err != nil {
		return err
	}

	assetPath, err := store.EnsureAsset(owner, repo, rel.Tag, asset.Name, asset.DownloadURL)
	if err != nil {
		return err
	}

	destDir := filepath.Join(store.TagDir(owner, repo, rel.Tag), "unpacked")
	execTarget, err := archive.Prepare(assetPath, destDir, repo, *execPath)
	if err != nil {
		return err
	}

	return runner.Exec(execTarget, forwardArgs)
}

// resolveRelease returns the requested release, or the latest when tag is empty.
func resolveRelease(releases *release.Client, owner, repo, tag string) (release.Release, error) {
	if tag != "" {
		return releases.ByTag(owner, repo, tag)
	}
	return releases.Latest(owner, repo)
}

// indexOf returns the index of value in args, or -1 when absent.
func indexOf(args []string, value string) int {
	for i, a := range args {
		if a == value {
			return i
		}
	}
	return -1
}

// splitRepo splits an "org/repo" argument into its owner and repo parts.
func splitRepo(arg string) (string, string, error) {
	owner, repo, ok := strings.Cut(arg, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return "", "", fmt.Errorf("expected <org/repo>, got %q", arg)
	}
	return owner, repo, nil
}
