BUILD_DIR?=$(shell pwd)/build

.PHONY: crosscompile
crosscompile: ## @build Cross-compile beat for the OS'es specified in GOX_OS variable. The binaries are placed in the build/bin directory.
	mkdir -p ${BUILD_DIR}/bin
	GOOS=linux GOARCH=amd64 go build -o  ${BUILD_DIR}/bin/cvmart_log_pilot_linux_amd64
	GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -o ${BUILD_DIR}/bin/cvmart_log_pilot_linux_arm64
#
#
#	gox -output="${BUILD_DIR}/bin/{{.Dir}}-{{.OS}}-{{.Arch}}" -osarch "linux/amd64 linux/arm64 linux/arm"