
########################################################################################################################
# Copyright (c) 2020 IoTeX
# This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
# warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
# permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
# License 2.0 that can be found in the LICENSE file.
########################################################################################################################

NAME=iotex/iotex-analyser
# Go parameters
GOCMD=go
GOLINT=golint
GOBUILD=$(GOCMD) build -tags 'nosilkworm,purego' -ldflags="-s -w"
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
PLUGIN_DIRECTORIES = $(wildcard plugins/*)
PLUGIN_SOS = $(notdir $(addsuffix .so,$(PLUGIN_DIRECTORIES)))

.PHONY: plugins plugin build proto all run

all:
	$(MAKE) clean
	$(MAKE) -j$(shell nproc) plugins
	$(MAKE) build

plugins: $(PLUGIN_SOS)

%.so:
	$(GOBUILD) -o $@ -buildmode=plugin plugins/$*/*.go


plugin:
	$(GOBUILD) -o $(name).so -buildmode=plugin plugins/$(name)/*.go

clean:
	rm -f *.so iotex-analyser
	
build:
	$(GOBUILD) -v .

dev:
	for plugin in $(PLUGIN_DIRECTORIES) ; do \
		so=`echo $${plugin}.so | sed 's/plugins\///g'` ; \
		go build -race -o $$so -buildmode=plugin $$plugin/*.go ; \
	done
	go build -race -v .

run: build

docker:
	docker build --progress=plain -t ${NAME}:latest  .