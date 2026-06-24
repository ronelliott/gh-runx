# Contributing

Thanks for your interest in contributing to `gh-runx`!

Contributions are welcome. This is a small project, so reviews may take some
time, and bug fixes are prioritized. Issues, ideas, and pull requests are all
appreciated.

## Features

For new features, please **open a Discussion** before creating a pull request.
This lets the change be thought through and discussed before any code is
written.

## Refactoring

If you are refactoring, include a brief explanation of the change and how it
improves the project.

## Bug Fixes

When reporting a bug — or opening a pull request that fixes one — include clear
steps to reproduce and the behavior before and after. A regression test is
preferred; reproduction steps are the minimum.

## Pull Request Hygiene

- Contributions must pass build, tests, and CI before merging. Run the full
  check locally first:

  ```sh
  test -z "$(gofmt -l .)" && go vet ./... && go build ./... && go test ./...
  ```

- Include tests or a minimal, reproducible example that demonstrates the change.
- Mark a pull request "Ready for Review" only when it is actually complete.

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0). This
keeps history readable and lets release automation generate the changelog and
version bumps.

## License

By contributing, you agree that you have authored 100% of the content (or have
the necessary rights to it) and that your contribution may be provided under the
project's [MIT license](LICENSE.md).
