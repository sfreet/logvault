# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder

WORKDIR /src

# Copy go module and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code
COPY . .

# Build the binary
# CGO_ENABLED=0 and GOOS=linux are for cross-compilation
# -ldflags="-s -w" strips debug information and symbols to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -o /logvault .

# Stage 2: Create the final, small image
FROM alpine:latest

WORKDIR /app

# Copy the static web files
COPY index.html login.html ./

# Copy the built binary from the builder stage
COPY --from=builder /logvault .

# Expose the ports the application listens on
# 8080 for web UI (HTTP/HTTPS)
# 514 for Syslog
EXPOSE 8080
EXPOSE 514/udp

# The config file, certs, and logs should be mounted as volumes.
# We expect config.yaml, server.crt, server.key to be in /app/ at runtime.

# Set the entrypoint for the container
CMD ["./logvault"]
