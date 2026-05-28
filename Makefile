.PHONY: ⚙
SHELL := bash

export GITEA_TOKEN = $(shell source .env; echo $$GITEA_TOKEN)

RPI_HOST   ?= pi400
RPI_USER   ?= uwe

# We use a Pi400 as dev machine for daily work.
# On this machine we skip ssh and install directly.
ON_PI := $(filter $(shell hostname),pi400)

addr            = $(RPI_USER)@$(RPI_HOST)
query_addr      = http://$(RPI_HOST):9101/metrics
node_query_addr = http://$(RPI_HOST):9100/metrics

upload_dir = /home/$(RPI_USER)/Downloads
run        = $(if $(ON_PI),,ssh -t -q $(addr))
copy       = $(if $(ON_PI),cp,scp -qp)
copy_dest  = $(if $(ON_PI),$(upload_dir),$(addr):$(upload_dir))
srcbin     = bin/rpi-exporter
testbin    = bin/rpi-exporter.test
name       = rpi-exporter

help: ⚙  ## show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*## "}; {printf "  %-16s %s\n", $$1, $$2}'

build-stub: ⚙  ## build vcgencmd stub for local testing
	go build -o bin/vcgencmd ./cmd/vcgencmd-stub

test: ⚙ build-stub  ## run tests locally with vcgencmd stub
	PATH="$(CURDIR)/bin:$(PATH)" go test ./...

.env:
	touch .env

snapshot: ⚙  ## build release archives for all platforms (no git tag required)
	goreleaser build --snapshot --clean

release: ⚙ .env  ## build and publish a release (requires a git tag and GITEA_TOKEN in .env)
	goreleaser release --clean

build: ⚙  ## cross-compile binary and test binary for arm64/linux
	GOARCH=arm64 GOOS=linux go build   -o "$(srcbin)"  ./cmd/rpi-exporter
	GOARCH=arm64 GOOS=linux go test -c -o "$(testbin)" ./collector/...

upload: ⚙ build  ## upload binaries to Raspberry Pi
	@echo "copying binary to $(addr):$(upload_dir)/"
	$(run) mkdir -p "$(upload_dir)"
	$(run) rm -f "$(upload_dir)/$(name)" "$(upload_dir)/$(name).test"
	$(copy) "$(srcbin)"  "$(copy_dest)/$(name)"
	$(copy) "$(testbin)" "$(copy_dest)/$(name).test"

test-uploads: ⚙ upload  ## build, upload, and run remote tests on Pi
	$(run) "$(upload_dir)/$(name)"      -rpi
	$(run) "$(upload_dir)/$(name).test" -test.v

install: ⚙ upload  ## install rpi-exporter as a systemd service on the Pi
	$(run) sudo $(upload_dir)/$(name) -install

uninstall: ⚙ upload  ## uninstall rpi-exporter systemd service from the Pi
	$(run) sudo $(upload_dir)/$(name) -uninstall

install-plugin: ⚙ upload  ## install rpi-exporter as a systemd timer (textfile plugin for node_exporter)
	$(run) sudo $(upload_dir)/$(name) -install-plugin

uninstall-plugin: ⚙ upload  ## uninstall rpi-exporter systemd timer plugin
	$(run) sudo $(upload_dir)/$(name) -uninstall-plugin

query: ⚙  ## query metrics endpoint on the Pi (standalone server mode)
	curl -k $(query_addr)

query-node: ⚙  ## query rpi metrics from node_exporter on the Pi (port 9100)
	curl -sk $(node_query_addr) | grep '^rpi_'

query-plugin: ⚙  ## query rpi metrics from node_exporter textfile output on the Pi
	$(run) grep '^rpi_' /var/lib/prometheus/node-exporter/rpi.prom

reuse: ⚙  ## check REUSE/SPDX license compliance
	reuse lint
