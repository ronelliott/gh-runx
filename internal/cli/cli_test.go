package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronelliott/gh-runx/internal/cli"
)

// TestParseFlagsBeforeRepo verifies that flags preceding the org/repo argument
// are parsed.
func TestParseFlagsBeforeRepo(t *testing.T) {
	opts, err := cli.Parse([]string{"--version", "v1.2.3", "octocat/tool"})
	require.NoError(t, err)
	require.Equal(t, "octocat", opts.Owner)
	require.Equal(t, "tool", opts.Repo)
	require.Equal(t, "v1.2.3", opts.Version)
}

// TestParseFlagsAfterRepo verifies that flags following the org/repo argument
// are parsed, the regression behind the --version-is-ignored bug.
func TestParseFlagsAfterRepo(t *testing.T) {
	opts, err := cli.Parse([]string{"octocat/tool", "--version", "v1.2.3"})
	require.NoError(t, err)
	require.Equal(t, "octocat", opts.Owner)
	require.Equal(t, "tool", opts.Repo)
	require.Equal(t, "v1.2.3", opts.Version)
}

// TestParseFlagsInterspersed verifies that flags on both sides of the org/repo
// argument are all parsed.
func TestParseFlagsInterspersed(t *testing.T) {
	opts, err := cli.Parse([]string{"--exec", "bin/app", "octocat/tool", "--force"})
	require.NoError(t, err)
	require.Equal(t, "tool", opts.Repo)
	require.Equal(t, "bin/app", opts.Exec)
	require.True(t, opts.Force)
}

// TestParseForceShorthand verifies that -f is an alias for --force.
func TestParseForceShorthand(t *testing.T) {
	opts, err := cli.Parse([]string{"octocat/tool", "-f"})
	require.NoError(t, err)
	require.True(t, opts.Force)
}

// TestParseForwardArgs verifies that arguments after "--" are forwarded and not
// interpreted as flags.
func TestParseForwardArgs(t *testing.T) {
	opts, err := cli.Parse([]string{"octocat/tool", "--version", "v1", "--", "--help", "sub"})
	require.NoError(t, err)
	require.Equal(t, "v1", opts.Version)
	require.Equal(t, []string{"--help", "sub"}, opts.ForwardArgs)
}

// TestParseDefaultsToLatest verifies that omitting --version leaves Version
// empty so the caller resolves the latest release.
func TestParseDefaultsToLatest(t *testing.T) {
	opts, err := cli.Parse([]string{"octocat/tool"})
	require.NoError(t, err)
	require.Empty(t, opts.Version)
	require.False(t, opts.Force)
}

// TestParseMissingRepoReturnsUsage verifies that no positional argument yields
// the usage error.
func TestParseMissingRepoReturnsUsage(t *testing.T) {
	_, err := cli.Parse([]string{"--force"})
	require.EqualError(t, err, cli.Usage)
}

// TestParseTooManyPositionalsReturnsUsage verifies that more than one positional
// argument yields the usage error.
func TestParseTooManyPositionalsReturnsUsage(t *testing.T) {
	_, err := cli.Parse([]string{"octocat/tool", "extra"})
	require.EqualError(t, err, cli.Usage)
}

// TestParseInvalidRepoReturnsError verifies that a malformed org/repo argument
// is rejected.
func TestParseInvalidRepoReturnsError(t *testing.T) {
	_, err := cli.Parse([]string{"notarepo"})
	require.Error(t, err)
	require.NotEqual(t, cli.Usage, err.Error())
}

// TestParseUnknownFlagReturnsError verifies that an unknown flag is rejected.
func TestParseUnknownFlagReturnsError(t *testing.T) {
	_, err := cli.Parse([]string{"octocat/tool", "--nope"})
	require.Error(t, err)
}
