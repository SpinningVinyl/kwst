.DEFAULT_GOAL := build

build: *.go
	go build -o build/kwst
