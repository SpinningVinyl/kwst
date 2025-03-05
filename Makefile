.DEFAULT_GOAL := build

build: *.go
	go build -o build/kwst -v -ldflags="-X 'main.Version=$$(git describe --always)' -X 'main.BuildTime=$$(date)'"
