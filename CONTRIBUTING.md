# Contributing to rentalot-cli

Thanks for your interest in contributing!

## Getting Started

```bash
git clone https://github.com/Rentalot-ai/rentalot-cli.git
cd rentalot-cli
make install   # Download dependencies
make build     # Build binary
make test      # Run tests
```

## Development

```bash
make build     # Build to bin/rentalot-cli
make test      # Run all tests
make lint      # Run linters
make format    # Format code
```

### CLI output golden tests

`make test` includes offline tests of the built CLI in `e2e/golden_9bh_test.go`.
They compare stdout and stderr byte for byte with the checked-in files in
`e2e/testdata/golden9bh/` and check the exit status. The cases cover help, table
and JSON output, empty lists, deletion confirmations, and argument/API errors.
API responses come from local `httptest` servers and schema-based JSON fixtures;
each CLI process uses an isolated home and configuration with a fake API key.
Only the temporary default config path in output becomes `<CONFIG_FILE>`;
spacing, Unicode, JSON indentation, and trailing newlines remain significant.

For an intentional output change, regenerate and review the affected files:

```bash
go test ./e2e -run '^TestGolden9bhCLI$' -update
git diff -- e2e/testdata/golden9bh
make test
make lint
```

Ordinary test runs never write golden files; missing or changed files fail.
To add a case, extend the table in `TestGolden9bhCLI`, supply a JSON fixture
when it calls the API, and generate its `.stdout.golden` and `.stderr.golden`
files with `-update`. Keep empty stream files so unexpected output also fails.

## Pull Requests

1. Fork the repo and create your branch from `main`
2. Add tests for any new functionality
3. Ensure `make test` and `make lint` pass
4. Update `CHANGELOG.yaml` with your changes

## Reporting Issues

Use [GitHub Issues](https://github.com/Rentalot-ai/rentalot-cli/issues). Include:
- What you expected vs what happened
- Steps to reproduce
- `rentalot-cli version` output
- OS and architecture

## Code Style

- Functions under 40 lines
- Errors wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Table-driven tests with `map[string]struct{}`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

