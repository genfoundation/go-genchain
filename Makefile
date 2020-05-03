# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: ggen android ios ggen-cross swarm evm all test clean
.PHONY: ggen-linux ggen-linux-386 ggen-linux-amd64 ggen-linux-mips64 ggen-linux-mips64le
.PHONY: ggen-linux-arm ggen-linux-arm-5 ggen-linux-arm-6 ggen-linux-arm-7 ggen-linux-arm64
.PHONY: ggen-darwin ggen-darwin-386 ggen-darwin-amd64
.PHONY: ggen-windows ggen-windows-386 ggen-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

ggen:
	build/env.sh go run build/ci.go install ./cmd/ggen
	@echo "Done building."
	@echo "Run \"$(GOBIN)/ggen\" to launch ggen."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/ggen.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Ggen.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

ggen-cross: ggen-linux ggen-darwin ggen-windows ggen-android ggen-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/ggen-*

ggen-linux: ggen-linux-386 ggen-linux-amd64 ggen-linux-arm ggen-linux-mips64 ggen-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-*

ggen-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/ggen
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-* | grep 386

ggen-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/ggen
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-* | grep amd64

ggen-linux-arm: ggen-linux-arm-5 ggen-linux-arm-6 ggen-linux-arm-7 ggen-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-* | grep arm

ggen-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/ggen
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-* | grep arm-5

ggen-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/ggen
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-* | grep arm-6

ggen-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/ggen
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-* | grep arm-7

ggen-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/ggen
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-* | grep arm64

ggen-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/ggen
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-* | grep mips

ggen-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/ggen
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-* | grep mipsle

ggen-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/ggen
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-* | grep mips64

ggen-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/ggen
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/ggen-linux-* | grep mips64le

ggen-darwin: ggen-darwin-386 ggen-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/ggen-darwin-*

ggen-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/ggen
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/ggen-darwin-* | grep 386

ggen-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/ggen
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ggen-darwin-* | grep amd64

ggen-windows: ggen-windows-386 ggen-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/ggen-windows-*

ggen-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/ggen
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/ggen-windows-* | grep 386

ggen-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/ggen
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ggen-windows-* | grep amd64
