// How to use https://www.alexedwards.net/blog/how-to-manage-tool-dependencies-in-go-1.24-plus
module github.com/esacteksab/go-pretty-toml

go 1.26.0

tool (
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
	mvdan.cc/gofumpt
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/exp/typeparams v0.0.0-20260212183809-81e46e3db34a // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/telemetry v0.0.0-20260213145524-e0ab670178e1 // indirect
	golang.org/x/tools v0.42.0 // indirect
	golang.org/x/vuln v1.1.4 // indirect
	honnef.co/go/tools v0.6.1 // indirect
	mvdan.cc/gofumpt v0.9.2 // indirect
)
