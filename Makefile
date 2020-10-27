.DEFAULT_GOAL := ./bin/takt

SRCS      := $(shell find . -type f -name '*.go' -not -name '*_test.go')

./bin/takt: $(SRCS)
	go build -o $@ ./cmd/takt

.PHONY: install
install:
	go install ./cmd/takt

.PHONY: clean
clean:
	rm -rf ./bin/*
