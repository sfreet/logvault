#!/bin/bash

COMPOSE_FILE="docker-compose.yaml"

if [ ! -f "$COMPOSE_FILE" ]; then
    echo "Error: $COMPOSE_FILE not found."
    exit 1
fi

case "$1" in
    start)
        echo "Starting Docker Compose services..."
        docker compose -f "$COMPOSE_FILE" up -d
        ;;
    stop)
        echo "Stopping Docker Compose services..."
        docker compose -f "$COMPOSE_FILE" down
        ;;
    restart)
        echo "Restarting Docker Compose services..."
        docker compose -f "$COMPOSE_FILE" down
        docker compose -f "$COMPOSE_FILE" up -d
        ;;
    build)
        echo "Building Docker images (logvault:latest)..."
        docker build -t logvault:latest .
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|build}"
        exit 1
        ;;
esac
