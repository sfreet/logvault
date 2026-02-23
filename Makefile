# Makefile for logvault

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GORUN=$(GOCMD) run

# Binary Name
BINARY_NAME=logvault

.PHONY: all build run clean test help docker docker-build package

all: build

# Build the application binary
build:
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -ldflags="-s -w" -o $(BINARY_NAME) .

# Build the docker image
docker-build: ## Build the main logvault docker image
	@echo "Building logvault:latest docker image..."
	@docker build -t logvault:latest .

# Alias for docker-build
docker: docker-build


# Run the application
run: 
	$(GORUN) .

# Clean the binary file
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f logvault.tar.gz ## Remove the offline package tarball

# Run tests (not yet implemented)
test:
	$(GOTEST) ./...

# Create the offline deployment package
package: docker-build ## Create the offline deployment package (logvault.tar.gz)
	@echo "Creating offline package: logvault.tar.gz"
	@mkdir -p logvault_package
	@echo "--> Saving Docker images..."
	@docker save -o logvault_package/logvault.tar logvault:latest
	@docker save -o logvault_package/redis.tar redis:7-alpine
	@echo "--> Copying configuration and scripts..."
	@cp docker-compose.yaml docker-compose.sh server.crt server.key logvault_package/
	@cp config.yaml.example logvault_package/config.yaml
	@echo '#!/bin/bash' > logvault_package/load_images.sh
	@echo 'echo "Loading Docker images..."' >> logvault_package/load_images.sh
	@echo 'docker load -i logvault.tar' >> logvault_package/load_images.sh
	@echo 'docker load -i redis.tar' >> logvault_package/load_images.sh
	@echo 'echo "Images loaded."' >> logvault_package/load_images.sh
	@chmod +x logvault_package/load_images.sh
	@echo 'Offline Package for Logvault\n\nInstructions:\n1. Un-tar this package: tar -xzvf logvault.tar.gz\n2. Go into the '\''logvault_package'\'' directory: cd logvault_package\n3. Load the Docker images: ./load_images.sh\n4. Edit the '\''config.yaml'\'' file to match your environment.\n5. Make the control script executable (if it isn'\''t already): chmod +x docker-compose.sh\n6. Start the services: ./docker-compose.sh start\n\nTo stop the services: ./docker-compose.sh stop\nTo restart the services: ./docker-compose.sh restart' > logvault_package/README_OFFLINE.txt
	@echo "--> Creating tarball..."
	@tar -czvf logvault.tar.gz logvault_package
	@echo "--> Cleaning up..."
	@rm -rf logvault_package
	@echo " "
	@echo "Package logvault.tar.gz created successfully."


# Help: Show available commands
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%%-10s\033[0m %%s\n", $$1, $$2}'
