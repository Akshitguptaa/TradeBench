module github.com/tradebench/tests

go 1.26.3

require github.com/stretchr/testify v1.11.1

require (
	github.com/segmentio/kafka-go v0.4.47
	github.com/tradebench/dummy-orderbook v0.0.0-00010101000000-000000000000
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/tradebench/dummy-orderbook => ../apps/dummy-orderbook
