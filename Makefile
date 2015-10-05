SRC := $(wildcard *.go)
TARGET := gxproxy

all: $(TARGET)

complexity: $(SRC)
	gocyclo -over 10 $(SRC)

lint: $(SRC)
	golint $(SRC)

test: $(SRC) lint complexity
	go test -v ./...

$(TARGET): $(SRC) lint complexity test
	go build -o $@

clean:
	$(RM) $(TARGET)

.PHONY: clean
