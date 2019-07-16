all: vet lint test build

.PHONY: build
build:
	bin/build

build-image: build
	 bin/build-image

test: test-unit

gen-fakes:
	bin/gen-fakes

vet:
	bin/vet

lint:
	bin/lint

test-unit:
	bin/test-unit

.PHONY: tools
tools:
	bin/tools
