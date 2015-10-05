SRC := $(wildcard *.go)
TARGET := gxproxy

all: $(TARGET)

complexity: $(SRC)
	gocyclo -over 10 $(SRC)

$(TARGET): $(SRC) lint complexity
	go build -o $@

lint: $(SRC)
	golint $(SRC)

clean:
	$(RM) $(TARGET)

.PHONY: clean
