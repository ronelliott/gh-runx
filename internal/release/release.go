// Package release looks up GitHub repository releases and selects the asset
// matching the running platform.
package release

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Asset is a single downloadable file attached to a release.
type Asset struct {
	// DownloadURL is the browser download URL for the asset.
	DownloadURL string

	// Name is the asset file name.
	Name string
}

// Release is a published repository release.
type Release struct {
	// Assets are the files attached to the release.
	Assets []Asset

	// Tag is the release tag name.
	Tag string
}

// getter fetches and decodes a REST API path into response.
type getter interface {
	Get(path string, response interface{}) error
}

// Client looks up repository releases through the GitHub REST API.
type Client struct {
	get getter
}

// New constructs a Client backed by the given REST getter.
func New(get getter) (*Client, error) {
	if get == nil {
		return nil, errors.New("release: nil REST getter")
	}
	return &Client{get: get}, nil
}

// ByTag returns the release matching tag for owner/repo.
func (c *Client) ByTag(owner, repo, tag string) (Release, error) {
	return c.fetch(fmt.Sprintf("repos/%s/%s/releases/tags/%s", owner, repo, tag))
}

// Latest returns the latest published release for owner/repo.
func (c *Client) Latest(owner, repo string) (Release, error) {
	return c.fetch(fmt.Sprintf("repos/%s/%s/releases/latest", owner, repo))
}

// fetch retrieves and converts the release at the given API path.
func (c *Client) fetch(path string) (Release, error) {
	var payload releasePayload
	if err := c.get.Get(path, &payload); err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return Release{}, ErrReleaseNotFound
		}
		return Release{}, fmt.Errorf("release: fetching %s: %w", path, err)
	}
	return payload.toRelease(), nil
}

// releasePayload mirrors the GitHub release JSON shape.
type releasePayload struct {
	Assets []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
		Name               string `json:"name"`
	} `json:"assets"`

	TagName string `json:"tag_name"`
}

// toRelease converts the JSON payload to a Release.
func (p releasePayload) toRelease() Release {
	assets := make([]Asset, 0, len(p.Assets))
	for _, a := range p.Assets {
		assets = append(assets, Asset{DownloadURL: a.BrowserDownloadURL, Name: a.Name})
	}
	return Release{Assets: assets, Tag: p.TagName}
}
