SRC := $(wildcard *.go)
TARGET := gxproxy

all: $(TARGET)

dep_lint:
	go get github.com/golang/lint/golint

dep_complex:
	go get github.com/fzipp/gocyclo

deps:
	go get github.com/op/go-logging
	go get github.com/fsouza/go-dockerclient

complexity: $(SRC) dep_complex
	gocyclo -over 10 $(SRC)

lint: $(SRC) dep_lint
	golint $(SRC)

test: $(SRC) lint complexity deps
	go test -v ./...

$(TARGET): $(SRC) lint complexity test deps
	go build -o $@

clean:
	$(RM) $(TARGET)

.PHONY: clean
