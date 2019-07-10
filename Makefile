all: test build

.PHONY: build
build:
	bin/build

test: vet lint test-unit

vet:
	bin/vet

lint:
	bin/lint

test-unit:
	bin/test-unit
