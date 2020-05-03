.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/genchain/go-genchain/cmd/swarm
	go build -o bin/ggen     github.com/genchain/go-genchain/cmd/ggen
	go build -o bin/bootnode github.com/genchain/go-genchain/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/genchain/go-genchain/swarm/...
