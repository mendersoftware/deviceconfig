GO ?= go
GOFMT ?= gofmt
DOCKER ?= docker
PACKAGES ?= $(shell go list ./... | \
				grep -v '\(vendor\|mock\|test\)' | \
				tr  '\n' ,)
DOCFILES := $(wildcard docs/*_api.yml)
ROOTDIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

DOCKERTAG ?= $(shell git rev-parse --abbrev-ref HEAD)
DOCKERTESTTAG ?= prtest
DOCKERIMAGE ?= mendersoftware/deviceconfig

GOFILES := $(shell find . -name "*.go" -type f -not -path './vendor/*')
SRCFILES := $(filter-out _test.go,$(GOFILES))

BINFILE := bin/deviceconfig
COVERFILE := coverage.txt

.PHONY: all
all: fmt lint docs build

tests/%: docs/%.yml
	[ -e $@ ] && rm -r $@; \
	docker run --rm -t -v $(ROOTDIR):$(ROOTDIR) -w $(ROOTDIR) \
		-u $(shell id -u):$(shell id -g) \
		openapitools/openapi-generator-cli:v4.3.1 generate \
		-g python -i $< \
		-c tests/.openapi-generator.yml \
		-o $(dir $@) \
		--additional-properties=packageName=$*

$(BINFILE): $(SRCFILES)
	$(GO) build -o $@ .

$(BINFILE).test: $(GOFILES)
	go test -c -o $(BINFILE).test \
		-cover -covermode atomic \
		-coverpkg $(PACKAGES)

$(COVERFILE): $(GOFILES)
	$(GO) test -cover -covermode=atomic -coverprofile=$@ ${TEST_FLAGS} ./...

.PHONY: docs
docs: $(patsubst docs/%.yml,tests/%,$(DOCFILES))

.PHONY: build
build: $(BINFILE)

.PHONY: build-test
build-test: $(BINFILE).test

.PHONY: test
test: $(COVERFILE)

# Dockerfile targets
bin/deviceconfig.docker: Dockerfile $(SRCFILES)
	docker rmi $(DOCKERIMAGE):$(DOCKERTAG) 2>/dev/null; \
	docker build . -f Dockerfile -t $(DOCKERIMAGE):$(DOCKERTAG)
	docker save $(DOCKERIMAGE):$(DOCKERTAG) -o $@

bin/deviceconfig.acceptance.docker: Dockerfile.acceptance $(GOFILES)
	docker rmi $(DOCKERIMAGE):$(DOCKERTESTTAG) 2>/dev/null; \
	docker build . -f Dockerfile.acceptance -t $(DOCKERIMAGE):$(DOCKERTESTTAG)
	docker save $(DOCKERIMAGE):$(DOCKERTESTTAG) -o $@

bin/acceptance.docker: tests/Dockerfile tests/requirements.txt
	docker rmi tests 2>/dev/null; \
	docker build tests -f tests/Dockerfile -t testing
	docker save testing -o $@

.PHONY: docker
docker: bin/deviceconfig.docker

.PHONY: docker-test
docker-test: bin/deviceconfig.acceptance.docker

.PHONY: docker-acceptance
docker-acceptance: bin/acceptance.docker

.PHONY: acceptance-tests
acceptance-tests: docker-acceptance docker-test docs
	docker-compose \
		-f tests/docker-compose.yml \
		-p acceptance \
		up -d
	docker attach acceptance_tester_1

.PHONY: acceptance-tests-logs
acceptance-tests-logs:
	for service in $(shell docker-compose -f tests/docker-compose.yml -p acceptance ps -a --services); do \
		docker-compose -p acceptance -f tests/docker-compose.yml \
				logs --no-color $$service > "tests/acceptance.$${service}.logs"; \
	done

.PHONY: acceptance-tests-down
acceptance-tests-down:
	docker-compose \
		-f tests/docker-compose.yml \
		-p acceptance down


.PHONY: fmt
fmt:
	$(GOFMT) -s -w $(GOFILES)

.PHONY: lint
lint:
	golangci-lint run -v

.PHONY: clean
clean:
	$(GO) clean -modcache -x -i ./...
	find . -name coverage.txt -delete
	rm -f tests/acceptance.*.logs tests/results.xml \
		tests/coverage-acceptance.txt coverage.txt
	rm -f bin/*
