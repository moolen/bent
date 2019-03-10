.DEFAULT_GOAL	:= build

#------------------------------------------------------------------------------
# Variables
#------------------------------------------------------------------------------

SHELL 	:= /bin/bash
BINDIR	:= bin
PKG 		:= github.com/moolen/bent
GOFILES		= $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./envoy/*")

.PHONY: build
build: vendor
	@echo "--> building"
	CGO_ENABLED=0 go build -o ./bin/bent cmd/bent/main.go
	CGO_ENABLED=0 go build -o ./bin/metadata cmd/envoy/metadata.go
	CGO_ENABLED=0 go build -o ./bin/trace-fwd cmd/trace-fwd/main.go

.PHONY: clean
clean:
	@echo "--> cleaning compiled objects and binaries"
	@go clean -tags netgo -i ./...
	@rm -rf $(BINDIR)/*

.PHONY: test
test: vendor
	@echo "--> running unit tests"
	@go test ./pkg/...

.PHONY: cover
cover:
	@echo "--> running coverage tests"
	go test -v -cover ./pkg/...

.PHONY: check
check: vet lint

.PHONY: vet
vet: tools.govet
	@echo "--> checking code correctness with 'go vet' tool"
	@go vet ./pkg/...

.PHONY: lint
lint: tools.golint
	@echo "--> checking code style with 'golint' tool"
	@echo golint ./pkg/...

docker: vendor build
	@echo "--> building docker image"
	$(SHELL) build/run.sh

docker.release: docker
	docker push moolen/bent-envoy:latest
	docker push moolen/trace-fwd:latest
	docker push moolen/bent:latest

#-----------------
#-- code generaion
#-----------------

generate: $(BINDIR)/gogofast $(BINDIR)/validate
	@echo "--> generating pb.go files"
	$(SHELL) build/generate_protos.sh

#------------------
#-- dependencies
#------------------
.PHONY: depend.update depend.install

depend.update: tools.glide
	@echo "--> updating dependencies from glide.yaml"
	@glide update

depend.install: tools.glide
	@echo "--> installing dependencies from glide.lock "
	@glide install

vendor:
	@echo "--> installing dependencies from glide.lock "
	@glide install

$(BINDIR):
	@mkdir -p $(BINDIR)

#---------------
#-- tools
#---------------
.PHONY: tools tools.glide tools.golint tools.govet

tools: tools.glide tools.golint tools.govet

tools.govet:
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		echo "--> installing govet"; \
		go get golang.org/x/tools/cmd/vet; \
	fi

tools.golint:
	@command -v golint >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing golint"; \
		go get -u golang.org/x/lint/golint; \
	fi

tools.glide:
	@command -v glide >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing glide"; \
		curl https://glide.sh/get | sh; \
	fi

$(BINDIR)/gogofast: vendor
	@echo "--> building $@"
	@go build -o $@ vendor/github.com/gogo/protobuf/protoc-gen-gogofast/main.go

$(BINDIR)/validate: vendor
	@echo "--> building $@"
	@go build -o $@ vendor/github.com/lyft/protoc-gen-validate/main.go
