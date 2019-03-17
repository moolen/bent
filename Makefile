.DEFAULT_GOAL	:= build

#------------------------------------------------------------------------------
# Variables
#------------------------------------------------------------------------------

SHELL 	:= /bin/bash
BINDIR	:= bin
PKG 		:= github.com/moolen/bent
GOFILES		= $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./envoy/*")

.PHONY: build
build: vendor envoy
	@echo "--> building"
	CGO_ENABLED=0 go build -o ./bin/bent cmd/bent/main.go
	CGO_ENABLED=0 go build -o ./bin/metadata cmd/envoy/metadata.go
	CGO_ENABLED=0 go build -o ./bin/trace-fwd cmd/trace-fwd/main.go

.PHONY: clean
clean:
	@echo "--> cleaning compiled objects and binaries"
	@go clean -tags netgo -i ./...
	@rm -rf $(BINDIR)
	@rm -rf envoy
	@rm -rf vendor

.PHONY: test
test: vendor envoy check
	@echo "--> running unit tests"
	@go test ./pkg/...

.PHONY: cover
cover:
	@echo "--> running coverage tests"
	go test -v -cover ./pkg/...

.PHONY: check
check: vendor envoy vet lint staticcheck

.PHONY: staticcheck
staticcheck: $(BINDIR)/staticcheck
	@echo "--> running static checks $@"
	$(BINDIR)/staticcheck ./pkg/...

.PHONY: vet
vet: tools.govet
	@echo "--> checking code correctness with 'go vet' tool"
	@go vet ./pkg/...

.PHONY: lint
lint: tools.golint
	@echo "--> checking code style with 'golint' tool"
	@echo golint ./pkg/...

docker:
	@echo "--> building docker image"
	$(SHELL) build/run.sh

docker.release: docker
	@echo "--> push docker images $@"
	docker push moolen/bent:${TRAVIS_TAG}
	docker push moolen/bent-envoy:${TRAVIS_TAG}
	docker push moolen/bent-trace-fwd:${TRAVIS_TAG}
	docker push moolen/envoy-authz:${TRAVIS_TAG}
	docker push moolen/envoy-echo:${TRAVIS_TAG}
	docker push moolen/envoy-egress:${TRAVIS_TAG}
	docker push moolen/envoy-fwd:${TRAVIS_TAG}

#-----------------
#-- code generaion
#-----------------
envoy: $(BINDIR)/gogofast $(BINDIR)/validate $(BINDIR)/protoc
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
	@echo "--> creating bin/ dir"
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

$(BINDIR)/gogofast: vendor $(BINDIR)
	@echo "--> building $@"
	@go build -o $@ vendor/github.com/gogo/protobuf/protoc-gen-gogofast/main.go

$(BINDIR)/validate: vendor $(BINDIR)
	@echo "--> building $@"
	@go build -o $@ vendor/github.com/lyft/protoc-gen-validate/main.go

$(BINDIR)/staticcheck: vendor $(BINDIR)
	@echo "--> installing staticcheck $@"
	curl -o $(BINDIR)/staticcheck -sL https://github.com/dominikh/go-tools/releases/download/2019.1.1/staticcheck_linux_amd64
	chmod +x $(BINDIR)/staticcheck

$(BINDIR)/protoc: $(BINDIR)
	@echo "--> fetching protoc $@"
	@go get -v -d google.golang.org/grpc
	@go get -v -d -t github.com/golang/protobuf/...
	mkdir -p $(BINDIR)/protoc.tmp
	curl -o $(BINDIR)/protoc.zip -sL https://github.com/google/protobuf/releases/download/v3.6.1/protoc-3.6.1-linux-x86_64.zip
	unzip -o $(BINDIR)/protoc.zip -d $(BINDIR)/protoc.tmp
	mv $(BINDIR)/protoc.tmp/bin/protoc $(BINDIR)/protoc
	@rm -rf $(BINDIR)/protoc.tmp $(BINDIR)/protoc.zip
	touch -m $(BINDIR)/protoc
