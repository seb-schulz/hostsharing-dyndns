GO ?= $(shell which go)
VERSION ?= $(shell git describe --tags --abbrev --always)

-include Makefile.variables

export

.PHONY: help
help:		          ## Display this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

.PHONY: build
build:                    ## Bundle binary
	CGO_ENABLED=0 $(GO) $@ -v -ldflags '-w -extldflags "-static"' ./...

.PHONY: test vet
test vet:
	$(GO) $@ ./...

.PHONY: clean
clean:                    ## Tidy up binary
	$(GO) $@ ./...

.PHONY: release
release:                  ## Create release page on Github and upload distribution package
	./scripts/$@.sh
