GOOS ?= linux
GOARCH ?= amd64
SRC := $(wildcard *.go)
TARGET := gxproxy-${GOOS}-${GOARCH}

all: $(TARGET)

deps:
	go get github.com/golang/lint/golint
	go get github.com/fzipp/gocyclo
	go get github.com/op/go-logging
	go get github.com/fsouza/go-dockerclient
	go get github.com/codegangsta/cli

complexity: $(SRC) deps
	gocyclo -over 10 $(SRC)

lint: $(SRC) complexity deps
	golint $(SRC)

test: $(SRC) deps
	go test -v ./...

$(TARGET): $(SRC) deps
	go build -o $@

clean:
	$(RM) $(TARGET)

.PHONY: clean
