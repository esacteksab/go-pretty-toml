# go-pretty-toml

A lightweight TOML formatter that aligns values and provides optional indentation for improved readability.

## Motivation

I wanted pretty TOML. I wanted Go.

## Overview

`go-pretty-toml` is a command-line utility that formats TOML files with consistent alignment and optional indentation. It preserves all data values while making your configuration files more readable and maintainable, **_but does not preserve comments_**.

Key features:

- Aligns values for clean, readable formatting
- Optional two-space indentation
- Sorts keys alphabetically
- Preserves data types
- Handles nested tables and array tables properly
- In-place file editing or stdout output

## Installation

```bash
go install github.com/esacteksab/go-pretty-toml@latest
```

## Usage

### Basic Usage

Format a file and print to stdout:

```bash
toml-fmt config.toml
```

Format a file with indentation:

```bash
toml-fmt -i config.toml
```

Format a file in-place (overwrite original):

```bash
toml-fmt -w config.toml
```

Format from stdin:

```bash
cat config.toml | toml-fmt
```

### Command-line Options

- `-w, --write`: Write result back to source file instead of stdout
- `-i, --indent`: Indent output using two spaces
- `-h, --help`: Show help

## Examples

### Before Formatting

```toml
# This is a TOML document

title = "TOML Example"

[owner]
name = "Tom Preston-Werner"
dob = 1979-05-27T07:32:00-08:00

[database]
enabled = true
ports = [ 8000, 8001, 8002 ]
data = [ ["delta", "phi"], [3.14] ]
temp_targets = { cpu = 79.5, case = 72.0 }

[servers]

[servers.alpha]
ip = "10.0.0.1"
role = "frontend"

[servers.beta]
ip = "10.0.0.2"
role = "backend"
```

### After Formatting (without `-i` flag)

```toml
title = "TOML Example"

[database]
data    = [["delta", "phi"], [3.14]]
enabled = true
ports   = [8000, 8001, 8002]

[database.temp_targets]
case = 72
cpu  = 79.5

[owner]
dob  = 1979-05-27T07:32:00-08:00
name = "Tom Preston-Werner"

[servers]

[servers.alpha]
ip   = "10.0.0.1"
role = "frontend"

[servers.beta]
ip   = "10.0.0.2"
role = "backend"
```

### After Formatting (with `-i` flag)

```toml
title = "TOML Example"

[database]
  data    = [["delta", "phi"], [3.14]]
  enabled = true
  ports   = [8000, 8001, 8002]

  [database.temp_targets]
    case = 72
    cpu  = 79.5

[owner]
  dob  = 1979-05-27T07:32:00-08:00
  name = "Tom Preston-Werner"

[servers]

  [servers.alpha]
    ip   = "10.0.0.1"
    role = "frontend"

  [servers.beta]
    ip   = "10.0.0.2"
    role = "backend"
```

> [!WARNING]
> The formatter currently does not preserve comments in TOML files. Any comments in the source file will be removed during formatting.

## How It Works

The formatter:

1. Parses TOML into a structured map
1. Categorizes keys into simple key-value pairs, tables, and array tables
1. Sorts keys alphabetically within each category
1. Formats each section with proper alignment and indentation
1. Writes the formatted output

## Integration

`go-pretty-toml` can be integrated into your CI/CD pipeline to enforce consistent TOML formatting. For example, with GitHub Actions:

```yaml
- name: Format TOML files
  run: |
    go install github.com/esacteksab/go-pretty-toml@latest
    find . -name "*.toml" -exec toml-fmt -w -i {} \;
```

## License

MIT License
