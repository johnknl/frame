module github.com/johnknl/frame

go 1.26.8

require github.com/stretchr/testify v1.12.1

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/johnknl/go-tools v0.0.2 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

tool (
	github.com/johnknl/go-tools/cmd/bench
	github.com/johnknl/go-tools/cmd/docs
	github.com/johnknl/go-tools/cmd/fuzz
	github.com/johnknl/go-tools/cmd/lint
	github.com/johnknl/go-tools/cmd/mut
	github.com/johnknl/go-tools/cmd/prof
	github.com/johnknl/go-tools/cmd/release
	github.com/johnknl/go-tools/cmd/test
	github.com/johnknl/go-tools/cmd/tools
)
