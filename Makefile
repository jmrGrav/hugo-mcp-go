SHELL := /bin/bash
.ONESHELL:

ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
SCRIPTS_DIR := $(ROOT)/scripts
DIST_DIR := $(ROOT)/dist

.PHONY: test race vet vuln secrets build build-all release-check clean

test:
	cd "$(ROOT)"
	go test ./...

race:
	cd "$(ROOT)"
	go test -race ./...

vet:
	cd "$(ROOT)"
	go vet ./...

vuln:
	cd "$(ROOT)"
	"$(SCRIPTS_DIR)/release-check.sh" --step vuln

secrets:
	cd "$(ROOT)"
	"$(SCRIPTS_DIR)/release-check.sh" --step secrets

build:
	cd "$(ROOT)"
	"$(SCRIPTS_DIR)/build-release.sh" local

build-all:
	cd "$(ROOT)"
	"$(SCRIPTS_DIR)/build-release.sh" all

release-check:
	cd "$(ROOT)"
	"$(SCRIPTS_DIR)/release-check.sh"

clean:
	cd "$(ROOT)"
	rm -rf "$(DIST_DIR)"
