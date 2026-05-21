.PHONY: ⚙

RPI_HOST   ?= pi4
RPI_USER   ?= uwe

addr       = $(RPI_USER)@$(RPI_HOST)
query_addr = http://$(RPI_HOST):9101/metrics
upload_dir = /home/$(RPI_USER)/Downloads
run        = ssh -q $(addr)
srcbin     = bin/rpi-exporter
testbin    = bin/rpi-exporter.test
name       = rpi-exporter

help: ⚙  ## show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*## "}; {printf "  %-16s %s\n", $$1, $$2}'

build-stub: ⚙  ## build vcgencmd stub for local testing
	go build -o bin/vcgencmd ./cmd/vcgencmd-stub

test: ⚙ build-stub  ## run tests locally with vcgencmd stub
	PATH="$(CURDIR)/bin:$(PATH)" go test -race ./...

build: ⚙  ## cross-compile binary and test binary for arm64/linux
	GOARCH=arm64 GOOS=linux go build   -o "$(srcbin)"  ./cmd/main.go
	GOARCH=arm64 GOOS=linux go test -c -o "$(testbin)" ./collector/...

upload: ⚙ build  ## upload binaries to Raspberry Pi
	@echo "copying binary to $(addr):$(upload_dir)/"
	$(run) mkdir -p "$(upload_dir)"
	$(run) rm -f "$(upload_dir)/$(name)" "$(upload_dir)/$(name).test"
	scp -qp "$(srcbin)"  "$(addr):$(upload_dir)/$(name)"
	scp -qp "$(testbin)" "$(addr):$(upload_dir)/$(name).test"

test-uploads: ⚙ upload  ## build, upload, and run remote tests on Pi
	$(run) "$(upload_dir)/$(name)"      -rpi
	$(run) "$(upload_dir)/$(name).test" -test.v

install: ⚙ upload  ## install rpi-exporter as a systemd service on the Pi
	$(run) sudo $(upload_dir)/$(name) -install

uninstall: ⚙ upload  ## uninstall rpi-exporter systemd service from the Pi
	$(run) sudo $(upload_dir)/$(name) -uninstall

query: ⚙  ## query metrics endpoint on the Pi
	curl -k $(query_addr)
