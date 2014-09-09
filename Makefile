.PHONY: all

VERSION := $(shell git describe --abbrev=0 --tags | sed 's/^v//')

all: build

build:
	go build -o bin/envredis -ldflags "-X main.Version $(VERSION)"
