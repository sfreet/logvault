# Makefile for logvault

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GORUN=$(GOCMD) run

# Binary Name
BINARY_NAME=logvault
HASH_TOOL_BINARY=bin/generate-password-hash
CONFIG_TOOL_BINARY=bin/configure-web-user
DIST_DIR=dist
DIST_BUNDLE_NAME=logvault-dist
DIST_WORKDIR=$(DIST_DIR)/$(DIST_BUNDLE_NAME)

.PHONY: all build build-hash-tool build-config-tool run clean test help docker docker-build package

all: build

# Build the application binary
build:
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -ldflags="-s -w" -o $(BINARY_NAME) .

build-hash-tool:
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=$$($(GOCMD) env GOARCH) $(GOBUILD) -a -ldflags="-s -w" -o $(HASH_TOOL_BINARY) ./cmd/generate-password-hash

build-config-tool:
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=$$($(GOCMD) env GOARCH) $(GOBUILD) -a -ldflags="-s -w" -o $(CONFIG_TOOL_BINARY) ./cmd/configure-web-user

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
	rm -f $(HASH_TOOL_BINARY)
	rm -f $(CONFIG_TOOL_BINARY)
	rm -f logvault.tar.gz ## Remove the offline package tarball
	rm -f logvault-dist.tar.gz
	rm -rf logvault-dist
	rm -rf $(DIST_DIR)

# Run tests (not yet implemented)
test:
	$(GOTEST) ./...

# Create the offline deployment package
package: docker-build build-hash-tool build-config-tool ## Create the offline deployment package (logvault.tar.gz)
	@echo "Creating offline package: logvault.tar.gz"
	@mkdir -p logvault_package/scripts logvault_package/bin
	@echo "--> Saving Docker images..."
	@docker save -o logvault_package/logvault.tar logvault:latest
	@docker save -o logvault_package/redis.tar redis:7-alpine
	@echo "--> Copying configuration and scripts..."
	@cp docker-compose.yaml compose.sh logvault_package/
	@cp .env.example logvault_package/.env.example
	@cp config.yaml.example logvault_package/config.yaml.example
	@cp scripts/generate_password_hash.sh logvault_package/scripts/
	@cp scripts/configure_web_user.sh logvault_package/scripts/
	@cp $(HASH_TOOL_BINARY) logvault_package/bin/
	@cp $(CONFIG_TOOL_BINARY) logvault_package/bin/
	@echo '#!/bin/bash' > logvault_package/load_images.sh
	@echo 'echo "Loading Docker images..."' >> logvault_package/load_images.sh
	@echo 'docker load -i logvault.tar' >> logvault_package/load_images.sh
	@echo 'docker load -i redis.tar' >> logvault_package/load_images.sh
	@echo 'echo "Images loaded."' >> logvault_package/load_images.sh
	@chmod +x logvault_package/load_images.sh logvault_package/scripts/generate_password_hash.sh logvault_package/scripts/configure_web_user.sh
	@echo 'Offline Package for Logvault\n\nThis directory is the extracted offline package.\n\nInstructions:\n1. If '\''config.yaml'\'' does not exist, create it from the example:\n   cp config.yaml.example config.yaml\n2. Review and edit '\''config.yaml'\'' to match your environment.\n3. Optional: if you need different host ports, edit '\''docker-compose.yaml'\'' before starting.\n4. In rootless Docker environments, avoid host ports below 1024.\n5. Optional: generate bcrypt password hashes with ./scripts/generate_password_hash.sh --password '\''your_secret'\''\n6. Optional: add or update a web user with ./scripts/configure_web_user.sh --config ./config.yaml --username admin --password '\''your_secret'\'' --role admin\n7. Load the Docker images: ./load_images.sh\n8. Start the services: ./compose.sh start\n\nUseful commands:\n- Stop the services: ./compose.sh stop\n- Restart the services: ./compose.sh restart\n- Reconfigure a web user: ./scripts/configure_web_user.sh --config ./config.yaml --username admin --password '\''your_secret'\'' --role admin' > logvault_package/README_OFFLINE.txt
	@echo "--> Creating tarball..."
	@tar -czvf logvault.tar.gz logvault_package
	@echo "--> Creating distribution bundle..."
	@rm -rf $(DIST_WORKDIR)
	@mkdir -p $(DIST_WORKDIR)
	@cp logvault.tar.gz $(DIST_WORKDIR)/
	@cp install-logvault.sh $(DIST_WORKDIR)/
	@chmod +x $(DIST_WORKDIR)/install-logvault.sh
	@tar -czvf logvault-dist.tar.gz -C $(DIST_DIR) $(DIST_BUNDLE_NAME)
	@echo "--> Cleaning up..."
	@rm -rf logvault_package
	@echo " "
	@echo "Package logvault.tar.gz created successfully."
	@echo "Distribution bundle created successfully: logvault-dist.tar.gz"


# Help: Show available commands
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%%-10s\033[0m %%s\n", $$1, $$2}'
