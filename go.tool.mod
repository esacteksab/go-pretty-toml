// How to use https://www.alexedwards.net/blog/how-to-manage-tool-dependencies-in-go-1.24-plus
module github.com/esacteksab/go-pretty-toml

go 1.27.4

tool (
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
	mvdan.cc/gofumpt
)

require (
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/pelletier/go-toml/v2 v2.4.3
	github.com/rogpeppe/go-internal v1.16.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	github.com/xhit/go-str2duration/v2 v2.2.0 // indirect
	golang.org/x/exp/typeparams v0.0.0-20261121134650-e1094eeca1b5 // indirect
	golang.org/x/mod v0.43.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/telemetry v0.0.0-20261119160235-508b12de1089 // indirect
	golang.org/x/tools v0.53.0 // indirect
	golang.org/x/vuln v1.10.0 // indirect
	honnef.co/go/tools v0.8.2 // indirect
	mvdan.cc/gofumpt v0.12.0 // indirect
)
