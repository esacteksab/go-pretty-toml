# 0.1.0 - 2025-04-22

## Changes by File

### `.mise.toml`

- Added pnpm tool version 10.8.0 configuration

### `.vscode/settings.json`

- Added file associations:
  - `go.tool.sum` associated with `Go-mod` icon
  - `go.tool.mod` associated with `Go-mod` icon
- Changed file association for `go.tool.mod` from `go.mod` to `go.sum`
- Added association for `go.tool.sum` to `go.sum`

### `internal/version/version.go`

- Uncommented and enabled the commit hash shortening logic (limiting to 7 characters)
- Added a linter directive `//nolint:mnd` to ignore magic number warning
- Removed several explanatory comments:
  - Comment about optionally shortening the commit hash
  - Comment indicating `info.Main` contains app info (simplified to "main module")
  - Comment about checksum being useful for verifying builds
  - Comment about potentially adding dependency info

### `golangci.yaml`

- Changed the `local-prefixes` for `goimports` from `github.com/esacteksab/gh-tp` to `github.com/esacteksab/go-pretty-toml`.

### `.goreleaser.yaml`

- Added `project_name: "toml-fmt"`.
- Changed `main` from `./cmd/tomlfmt` to `./cmd/toml-fmt` and added `binary: toml-fmt`.
- Updated the `ldflags` to point to the `internal/version` package instead of `cmd`.

### `README.md`

- Significant overhaul to provide a clearer description of the tool, its features, installation, usage, and examples.
- Added a warning about comment removal.
- Included a section on how the formatter works and how to integrate it into CI/CD.

### `internal/formatter/formatter.go`

- Added more detailed documentation to the functions including parameter descriptions and return values.
- Added comments to explain the purpose of various code sections.

### `testtoml/test1.toml`

- Refactored the TOML format from inline tables/maps within an array to an array of tables using `[[configs]]` for better readability and compatibility with the formatter.

### `testtoml/test2.toml`

- Minor changes to standardize the ordering of keys (`FileName`, `Name`, `Schemas`) within each `[[configs]]` array table.
