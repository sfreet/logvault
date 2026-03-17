#!/usr/bin/env bash

set -euo pipefail

BASE_DIR="${BASE_DIR:-/usr/geni}"
APP_DIR_NAME="${APP_DIR_NAME:-logvault}"
PACKAGE_NAME="${1:-logvault.tar.gz}"
TARGET_DIR="${BASE_DIR}/${APP_DIR_NAME}"
EXTRACTED_DIR="${BASE_DIR}/logvault_package"

if [[ ! -f "$PACKAGE_NAME" ]]; then
  echo "Error: package not found: $PACKAGE_NAME" >&2
  echo "Usage: $(basename "$0") [logvault.tar.gz]" >&2
  exit 1
fi

echo "Installing package to $TARGET_DIR ..."
mkdir -p "$BASE_DIR"
tar -xzf "$PACKAGE_NAME" -C "$BASE_DIR"

if [[ ! -d "$EXTRACTED_DIR" ]]; then
  echo "Error: extracted package directory not found: $EXTRACTED_DIR" >&2
  exit 1
fi

rm -rf "$TARGET_DIR"
mv "$EXTRACTED_DIR" "$TARGET_DIR"

if [[ -f "$TARGET_DIR/docker-compose.sh" ]]; then
  chmod +x "$TARGET_DIR/docker-compose.sh"
fi

if [[ -f "$TARGET_DIR/load_images.sh" ]]; then
  chmod +x "$TARGET_DIR/load_images.sh"
fi

if [[ -d "$TARGET_DIR/scripts" ]]; then
  find "$TARGET_DIR/scripts" -maxdepth 1 -type f -name "*.sh" -exec chmod +x {} \;
fi

if [[ ! -f "$TARGET_DIR/config.yaml" && -f "$TARGET_DIR/config.yaml.example" ]]; then
  cp "$TARGET_DIR/config.yaml.example" "$TARGET_DIR/config.yaml"
  echo "Created $TARGET_DIR/config.yaml from config.yaml.example"
fi

echo "Install completed."
echo "Next:"
echo "  cd $TARGET_DIR"
echo "  ./load_images.sh"
echo "  ./docker-compose.sh start"
