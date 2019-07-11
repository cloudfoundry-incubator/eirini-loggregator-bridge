all: vet lint test build

.PHONY: build
build:
	bin/build

test: test-unit

vet:
	bin/vet

lint:
	bin/lint

test-unit:
	bin/test-unit
