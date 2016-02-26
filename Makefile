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

gofmt: $(src)
	find $(SRC) -exec gofmt -w '{}' \;


lint: $(SRC) complexity deps gofmt
	golint $(SRC)

test: $(SRC) deps gofmt
	go test -v ./...

$(TARGET): $(SRC) deps gofmt
	go build -o $@

clean:
	$(RM) $(TARGET)

.PHONY: clean
