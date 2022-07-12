BINARY ?= fio
COUNT ?= 5
GOARCH = $(shell uname -s | tr '[:upper:]' '[:lower:]')
GOOS = $(shell uname -m)

.PHONY: all
all: run

.PHONY: run
run: $(BINARY)
	./$(BINARY) $(COUNT)

$(BINARY): main.go go.mod go.sum vendor Dockerfile
	docker build --build-arg GOOS=$(GOARCH) --build-arg GOARCH=$(GOOS) . --output .
	touch $(BINARY)

.PHONY: clean
clean:
	rm -f $(BINARY)