SRC := $(wildcard *.go)
TARGET := gxproxy

all: $(TARGET)

$(TARGET): $(SRC)
	go build -o $@

clean:
	$(RM) $(TARGET)

.PHONY: clean
