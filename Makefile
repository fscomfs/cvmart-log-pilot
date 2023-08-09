BUILD_DIR?=$(shell pwd)/build

.PHONY: crosscompile
crosscompile: ## @build Cross-compile beat for the OS'es specified in GOX_OS variable. The binaries are placed in the build/bin directory.
	mkdir -p ${BUILD_DIR}/bin
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o  ${BUILD_DIR}/bin/cvmart-daemon-linux-amd64
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -o ${BUILD_DIR}/bin/cvmart-daemon-linux-arm64