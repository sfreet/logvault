# Makefile for logvault

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GORUN=$(GOCMD) run

# Binary Name
BINARY_NAME=logvault

.PHONY: all build run clean test help docker docker-build

all: build

# Build the application binary
build:
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -ldflags="-s -w" -o $(BINARY_NAME) .

# Build the docker image
docker-build:
	docker build -t logvault .

# Alias for docker-build
docker: docker-build


# Run the application
run: 
	$(GORUN) .

# Clean the binary file
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Run tests (not yet implemented)
test:
	$(GOTEST) ./...

# Help: Show available commands
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%%-10s\033[0m %%s\n", $$1, $$2}'
