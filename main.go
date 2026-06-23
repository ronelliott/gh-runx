package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/ronelliott/gh-runx/internal/archive"
	"github.com/ronelliott/gh-runx/internal/cache"
	"github.com/ronelliott/gh-runx/internal/cli"
	"github.com/ronelliott/gh-runx/internal/release"
	"github.com/ronelliott/gh-runx/internal/runner"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gh-runx:", err)
		os.Exit(1)
	}
}

// run parses arguments, resolves and runs the matching release asset.
func run(args []string) error {
	opts, err := cli.Parse(args)
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

	rel, err := resolveRelease(releases, opts.Owner, opts.Repo, opts.Version)
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
	downloader, err := api.NewRESTClient(api.ClientOptions{
		Headers: map[string]string{"Accept": "application/octet-stream"},
	})
	if err != nil {
		return fmt.Errorf("creating asset download client: %w", err)
	}
	store, err := cache.New(root, downloader)
	if err != nil {
		return err
	}

	assetPath, err := store.EnsureAsset(opts.Owner, opts.Repo, rel.Tag, asset.Name, asset.URL, opts.Force)
	if err != nil {
		return err
	}

	destDir := filepath.Join(store.TagDir(opts.Owner, opts.Repo, rel.Tag), "unpacked")
	execTarget, err := archive.Prepare(assetPath, destDir, opts.Repo, opts.Exec, opts.Force)
	if err != nil {
		return err
	}

	return runner.Exec(execTarget, opts.ForwardArgs)
}

// resolveRelease returns the requested release, or the latest when tag is empty.
func resolveRelease(releases *release.Client, owner, repo, tag string) (release.Release, error) {
	if tag != "" {
		return releases.ByTag(owner, repo, tag)
	}
	return releases.Latest(owner, repo)
}
