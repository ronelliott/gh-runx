// Package cli parses the gh-runx command-line arguments into options the rest
// of the program consumes.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

// Usage is the one-line command synopsis.
const Usage = "usage: gh runx <org/repo> [--version tag] [--exec path] [--force] [-- args...]"

// Options holds the parsed command-line arguments.
type Options struct {
	// Exec is an explicit relative path to execute within an archive.
	Exec string

	// Force re-downloads and re-unpacks even when a cached copy exists.
	Force bool

	// ForwardArgs are the arguments after "--" passed through to the program.
	ForwardArgs []string

	// Owner is the repository owner.
	Owner string

	// Repo is the repository name.
	Repo string

	// Version is the release tag to run; empty means the latest release.
	Version string
}

// Parse converts raw command-line arguments into Options. Flags may appear
// before or after the org/repo argument; arguments after a "--" separator are
// forwarded to the executed program untouched.
func Parse(args []string) (Options, error) {
	var opts Options
	if idx := indexOf(args, "--"); idx >= 0 {
		opts.ForwardArgs = args[idx+1:]
		args = args[:idx]
	}

	fs := flag.NewFlagSet("gh-runx", flag.ContinueOnError)
	version := fs.String("version", "", "release tag to run (default: latest)")
	exec := fs.String("exec", "", "path within the archive to execute")
	force := fs.Bool("force", false, "re-download and re-unpack even if cached")
	fs.BoolVar(force, "f", false, "shorthand for --force")

	// flag.Parse stops at the first non-flag argument, so parse repeatedly to
	// collect positionals while still honoring flags that follow them.
	var positionals []string
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			return Options{}, err
		}
		if fs.NArg() == 0 {
			break
		}
		positionals = append(positionals, fs.Arg(0))
		rest = fs.Args()[1:]
	}

	if len(positionals) != 1 {
		return Options{}, errors.New(Usage)
	}

	owner, repo, err := splitRepo(positionals[0])
	if err != nil {
		return Options{}, err
	}

	opts.Exec = *exec
	opts.Force = *force
	opts.Owner = owner
	opts.Repo = repo
	opts.Version = *version
	return opts, nil
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
