
########################################################################################################################
# Copyright (c) 2020 IoTeX
# This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
# warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
# permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
# License 2.0 that can be found in the LICENSE file.
########################################################################################################################

# Go parameters
GOCMD=go
GOLINT=golint
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
PLUGIN_DIRECTORIES = $(dir $(wildcard plugins/*/*))

.PHONY: run

all : plugin build

plugin:
	for plugin in $(PLUGIN_DIRECTORIES) ; do \
		$(GOBUILD) -buildmode=plugin $$plugin/*.go ; \
	done

build:
	$(GOBUILD) -v .

run: build
