GOOS ?= linux
GOARCH ?= amd64
SRC := $(wildcard *.go)
TARGET := gxproxy-${GOOS}-${GOARCH}

all: $(TARGET)

deps:
	go install github.com/Masterminds/glide/...
	glide install

complexity: $(SRC) deps
	gocyclo -over 10 $(SRC)

vet: $(src) deps
	go vet

gofmt: $(src)
	find $(SRC) -exec gofmt -w '{}' \;

lint: $(SRC) deps
	golint $(SRC)

qc: lint vet complexity

test: $(SRC) deps gofmt
	go test -v ./...

$(TARGET): $(SRC) deps gofmt
	go build -o $@

clean:
	$(RM) $(TARGET)

.PHONY: clean
