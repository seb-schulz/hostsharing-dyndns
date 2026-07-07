GO ?= $(shell which go)
VERSION ?= $(shell git describe --tags --abbrev --always)

BUILD_ARGS ?= -v -ldflags '-w -extldflags "-static"'
BIN_NAME ?= hostsharing-dyndns
DESTDIR ?= .
DOMAIN ?= example.com
SSH_ARGS ?= -o UserKnownHostsFile=./ssh_known_hosts -o StrictHostKeyChecking=yes
SSH_HOST ?=
SCP_DEST ?= $(SSH_HOST):doms/$(DOMAIN)/fastcgi-ssl/

-include Makefile.variables

export

.PHONY: help
help:		          ## Display this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

.PHONY: build
build:                    ## Bundle binary
	CGO_ENABLED=0 $(GO) $@ $(BUILD_ARGS) -o $(DESTDIR)/$(BIN_NAME) ./...

.PHONY: deploy
deploy:              ## Deploy binary to hostsharing via SCP
ifneq ($(SSH_HOST),)
ifneq ($(SCP_DEST),)
	ssh $(SSH_ARGS) $(SSH_HOST) 'killall $(BIN_NAME)' || true
	scp $(SSH_ARGS) $(DESTDIR)/$(BIN_NAME) $(SCP_DEST)
else
	$(error "SCP_DEST undefined")
endif
else
	$(error "SSH_HOST undefined")
endif

.PHONY: test vet
test vet:
	$(GO) $@ ./...

.PHONY: clean
clean:                    ## Tidy up binary
	$(GO) $@ ./...
	rm -f $(DESTDIR)/$(BIN_NAME)

.PHONY: release
release:                  ## Create release page on Github and upload distribution package
	./scripts/$@.sh
