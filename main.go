package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/ronelliott/gh-runx/internal/archive"
	"github.com/ronelliott/gh-runx/internal/cache"
	"github.com/ronelliott/gh-runx/internal/checksum"
	"github.com/ronelliott/gh-runx/internal/cli"
	"github.com/ronelliott/gh-runx/internal/release"
	"github.com/ronelliott/gh-runx/internal/runner"
)

// requester fetches a URL through the authenticated go-gh REST client.
type requester interface {
	Request(method, path string, body io.Reader) (*http.Response, error)
}

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

	assetPath, downloaded, err := store.EnsureAsset(opts.Owner, opts.Repo, rel.Tag, asset.Name, asset.URL, opts.Force)
	if err != nil {
		return err
	}
	if downloaded {
		if err := verifyChecksum(downloader, rel.Assets, asset, assetPath); err != nil {
			return err
		}
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

// verifyChecksum verifies the downloaded asset against a SHA-256 sums manifest
// published in the release. It is a no-op when the release ships no manifest. A
// manifest that omits the asset is reported as a warning rather than a failure,
// since some releases sign only a subset of their assets.
func verifyChecksum(fetch requester, assets []release.Asset, asset release.Asset, path string) error {
	var manifest release.Asset
	found := false
	for _, a := range assets {
		if checksum.IsManifest(a.Name) {
			manifest = a
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	body, err := fetchAsset(fetch, manifest.URL)
	if err != nil {
		return fmt.Errorf("verifying %s: %w", asset.Name, err)
	}

	want, ok := checksum.Lookup(string(body), asset.Name)
	if !ok {
		fmt.Fprintf(os.Stderr, "gh-runx: warning: %s not listed in %s; skipping checksum verification\n", asset.Name, manifest.Name)
		return nil
	}

	if err := checksum.Verify(path, want); err != nil {
		// Drop the unverified file so a later run re-downloads instead of
		// trusting the cached copy.
		os.Remove(path)
		return err
	}
	return nil
}

// fetchAsset downloads the bytes at url through the authenticated client.
func fetchAsset(fetch requester, url string) ([]byte, error) {
	resp, err := fetch.Request(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: unexpected status %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
