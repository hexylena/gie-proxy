GOOS ?= linux
GOARCH ?= amd64
SRC := $(wildcard *.go)
TARGET := gxproxy-${GOOS}-${GOARCH}

all: $(TARGET)

deps:
	go get github.com/Masterminds/glide/...
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

coverage.txt: $(SRC)
	go test -coverprofile coverage.txt $(glide novendor)

qc: lint vet complexity

test: $(SRC) deps gofmt
	go test -v $(glide novendor)

$(TARGET): $(SRC) deps gofmt
	go build -o $@

clean:
	$(RM) $(TARGET)

.PHONY: clean
