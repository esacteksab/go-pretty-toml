## Motivation

> [!WARNING]
> Naming is hard. While the repo is currently called `go-pretty-toml`, I'm not sure what to call the binary. `tomlfmt`, `toml-fmt`, `go-pretty-toml`, `goprettytoml`. Until then, I likely won't cut a release.

I wanted pretty TOML. I wanted Go. Sorts tables and their keys alphabetically. Supports indenting 2 spaces with `-i`. Will overwrite existing file with `-w`. Can read from `stdin`.

Given a TOML file:

```toml
name = "Test Config"

[[database]]
  host  = "db1.example.com"
  creds = "user:pass"

        [client]
        timeout = 30

[server]
  ip   = "127.0.0.1"
  port = 8080

    [tls]
    enabled = true
    cert    = "/path/to/cert.pem"

[[database]]
  host  = "db2.example.com"
  creds = "admin:secret"

```

We get this

```bash

./tomlfmt test.toml

```

```toml

name = "Test Config"

[[database]]
creds = "user:pass"
host  = "db1.example.com"

[[database]]
creds = "admin:secret"
host  = "db2.example.com"

[client]
timeout = 30

[server]
ip   = "127.0.0.1"
port = 8080

[tls]
cert    = "/path/to/cert.pem"
enabled = true

```

By default, it will write to `stdout`. You can overwrite the file with `-w`. If you want things indented, you can pass `-i`.

```bash

./tomlfmt test.toml -i

```

```toml

name = "Test Config"

[[database]]
  creds = "user:pass"
  host  = "db1.example.com"

[[database]]
  creds = "admin:secret"
  host  = "db2.example.com"

[client]
  timeout = 30

[server]
  ip   = "127.0.0.1"
  port = 8080

[tls]
  cert    = "/path/to/cert.pem"
  enabled = true

```

It will also read from `stdin`

```bash

cat test.toml | ./toml-fmt -i

```

```toml

name = "Test Config"

[[database]]
  creds = "user:pass"
  host  = "db1.example.com"

[[database]]
  creds = "admin:secret"
  host  = "db2.example.com"

[client]
  timeout = 30

[server]
  ip   = "127.0.0.1"
  port = 8080

[tls]
  cert    = "/path/to/cert.pem"
  enabled = true

```

```bash
./tomlfmt -h
usage: toml-fmt [<flags>] [<filename>]

Formats TOML files with alignment and optional indentation.


Flags:
  -h, --[no-]help    Show context-sensitive help (also try --help-long and --help-man).
  -w, --[no-]write   Write result back to the source file instead of stdout.
  -i, --[no-]indent  Indent output using two spaces.

Args:
  [<filename>]  Input TOML file (optional, reads from stdin if omitted)
```
