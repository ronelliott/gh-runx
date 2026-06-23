package release_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/stretchr/testify/require"

	"github.com/ronelliott/gh-runx/internal/release"
)

// stubGetter is a getter that returns a canned body or error.
type stubGetter struct {
	body string
	err  error
}

// Get decodes the stub body into response, or returns the stub error.
func (s stubGetter) Get(path string, response interface{}) error {
	if s.err != nil {
		return s.err
	}
	return json.Unmarshal([]byte(s.body), response)
}

// TestClientLatestReturnsReleaseWithAssets verifies that Latest decodes the tag
// and assets from the REST payload.
func TestClientLatestReturnsReleaseWithAssets(t *testing.T) {
	body := `{
		"tag_name": "v1.2.3",
		"assets": [
			{"name": "tool_darwin_arm64.tar.gz", "browser_download_url": "https://example.com/a"}
		]
	}`
	client, err := release.New(stubGetter{body: body})
	require.NoError(t, err)

	rel, err := client.Latest("octocat", "tool")
	require.NoError(t, err)
	require.Equal(t, "v1.2.3", rel.Tag)
	require.Len(t, rel.Assets, 1)
	require.Equal(t, "tool_darwin_arm64.tar.gz", rel.Assets[0].Name)
	require.Equal(t, "https://example.com/a", rel.Assets[0].DownloadURL)
}

// TestClientByTagMapsNotFoundToSentinel verifies that a 404 from the API is
// translated to ErrReleaseNotFound.
func TestClientByTagMapsNotFoundToSentinel(t *testing.T) {
	client, err := release.New(stubGetter{err: &api.HTTPError{StatusCode: 404}})
	require.NoError(t, err)

	rel, err := client.ByTag("octocat", "tool", "v9.9.9")
	require.ErrorIs(t, err, release.ErrReleaseNotFound)
	require.Equal(t, release.Release{}, rel)
}

// TestClientLatestPropagatesOtherErrors verifies that non-404 errors are
// returned wrapped rather than as ErrReleaseNotFound.
func TestClientLatestPropagatesOtherErrors(t *testing.T) {
	sentinel := errors.New("boom")
	client, err := release.New(stubGetter{err: sentinel})
	require.NoError(t, err)

	_, err = client.Latest("octocat", "tool")
	require.ErrorIs(t, err, sentinel)
	require.NotErrorIs(t, err, release.ErrReleaseNotFound)
}

// TestNewRejectsNilGetter verifies that New returns an error for a nil getter.
func TestNewRejectsNilGetter(t *testing.T) {
	client, err := release.New(nil)
	require.Error(t, err)
	require.Nil(t, client)
}
