// How to use https://www.alexedwards.net/blog/how-to-manage-tool-dependencies-in-go-1.24-plus
module github.com/esacteksab/go-pretty-toml

go 1.24.3

tool (
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
	mvdan.cc/gofumpt
)

require (
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/exp/typeparams v0.0.0-20250506013437-ce4c2cf36ca6 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/telemetry v0.0.0-20250507143331-155ddd5254aa // indirect
	golang.org/x/tools v0.33.0 // indirect
	golang.org/x/vuln v1.1.4 // indirect
	honnef.co/go/tools v0.6.1 // indirect
	mvdan.cc/gofumpt v0.8.0 // indirect
)
