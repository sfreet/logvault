#!/usr/bin/env bash

set -euo pipefail

BASE_DIR="${BASE_DIR:-$HOME/opt}"
APP_DIR_NAME="${APP_DIR_NAME:-logvault}"
PACKAGE_NAME="${1:-logvault.tar.gz}"
TARGET_DIR="${BASE_DIR}/${APP_DIR_NAME}"
EXTRACTED_DIR="${BASE_DIR}/logvault_package"
CA_DIR="${CA_DIR:-$HOME/opt/opa/cert}"
CA_CERT="${CA_DIR}/myCA.crt"
CA_KEY="${CA_DIR}/myCA.key"
TLS_CERT="${TARGET_DIR}/server.crt"
TLS_KEY="${TARGET_DIR}/server.key"
CONFIG_FILE="${TARGET_DIR}/config.yaml"
HASH_TOOL="${TARGET_DIR}/bin/generate-password-hash"
ADMIN_HASH_PLACEHOLDER="__ADMIN_SECRET_HASH__"
OPS_HASH_PLACEHOLDER="__OPS_SECRET_HASH__"
API_TOKEN_PLACEHOLDER="__API_BEARER_TOKEN__"
TLS_CERT_CN="${TLS_CERT_CN:-Genian Logvault Server}"
TLS_CERT_O="${TLS_CERT_O:-Genians}"
TLS_CERT_IP="${TLS_CERT_IP:-}"

require_openssl() {
  if ! command -v openssl >/dev/null 2>&1; then
    echo "Error: openssl is required to generate TLS certificates." >&2
    exit 1
  fi
}

require_hash_tool() {
  if [[ ! -x "$HASH_TOOL" ]]; then
    echo "Error: password hash tool not found: $HASH_TOOL" >&2
    exit 1
  fi
}

escape_sed_replacement() {
  printf '%s' "$1" | sed -e 's/[\\/&]/\\&/g'
}

prompt_secret() {
  local prompt="$1"
  local secret=""
  local confirm=""

  while true; do
    read -rsp "${prompt}: " secret < /dev/tty
    echo >&2
    if [[ -z "$secret" ]]; then
      echo "Value cannot be empty." >&2
      continue
    fi
    read -rsp "Confirm ${prompt}: " confirm < /dev/tty
    echo >&2
    if [[ "$secret" != "$confirm" ]]; then
      echo "Values do not match. Try again." >&2
      continue
    fi
    printf '%s' "$secret"
    return 0
  done
}

prompt_value() {
  local prompt="$1"
  local default_value="${2:-}"
  local value=""

  if [[ -n "$default_value" ]]; then
    read -rp "${prompt} [${default_value}]: " value < /dev/tty
    echo >&2
    if [[ -z "$value" ]]; then
      value="$default_value"
    fi
  else
    read -rp "${prompt}: " value < /dev/tty
    echo >&2
  fi

  printf '%s' "$value"
}

is_ipv4_address() {
  local ip="$1"
  local octet

  if [[ ! "$ip" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]]; then
    return 1
  fi

  IFS='.' read -r -a octets <<< "$ip"
  for octet in "${octets[@]}"; do
    if ((octet < 0 || octet > 255)); then
      return 1
    fi
  done

  return 0
}

detect_default_tls_ip() {
  local detected_ip=""

  if command -v hostname >/dev/null 2>&1; then
    detected_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  fi

  if is_ipv4_address "$detected_ip"; then
    printf '%s' "$detected_ip"
  fi
}

resolve_tls_cert_ip() {
  local input_ip="${TLS_CERT_IP:-}"
  local default_ip=""

  if is_ipv4_address "$input_ip"; then
    printf '%s' "$input_ip"
    return 0
  fi

  default_ip="$(detect_default_tls_ip)"

  if [[ -t 0 ]]; then
    while true; do
      input_ip="$(prompt_value "Enter IPv4 address for TLS certificate SAN" "$default_ip")"
      if is_ipv4_address "$input_ip"; then
        printf '%s' "$input_ip"
        return 0
      fi
      echo "Invalid IPv4 address. Try again." >&2
    done
  fi

  if is_ipv4_address "$default_ip"; then
    printf '%s' "$default_ip"
    return 0
  fi

  echo "Error: TLS_CERT_IP is required for non-interactive installation when no default IPv4 address can be detected." >&2
  exit 1
}

ensure_initial_secrets() {
  local admin_password="${LOGVAULT_ADMIN_PASSWORD:-}"
  local ops_password="${LOGVAULT_OPS_PASSWORD:-}"
  local bearer_token="${LOGVAULT_API_BEARER_TOKEN:-}"
  local admin_hash=""
  local ops_hash=""
  local admin_hash_escaped=""
  local ops_hash_escaped=""
  local bearer_token_escaped=""

  if [[ ! -f "$CONFIG_FILE" ]]; then
    return 0
  fi

  if ! grep -qE "${ADMIN_HASH_PLACEHOLDER}|${OPS_HASH_PLACEHOLDER}|${API_TOKEN_PLACEHOLDER}" "$CONFIG_FILE"; then
    return 0
  fi

  require_hash_tool
  require_openssl

  if [[ -z "$admin_password" ]]; then
    if [[ -t 0 ]]; then
      admin_password="$(prompt_secret "Enter initial password for admin")"
    else
      echo "Error: LOGVAULT_ADMIN_PASSWORD is required for non-interactive installation." >&2
      exit 1
    fi
  fi

  if [[ -z "$ops_password" ]]; then
    if [[ -t 0 ]]; then
      ops_password="$(prompt_secret "Enter initial password for ops")"
    else
      echo "Error: LOGVAULT_OPS_PASSWORD is required for non-interactive installation." >&2
      exit 1
    fi
  fi

  if [[ -z "$bearer_token" ]]; then
    bearer_token="$(openssl rand -hex 32)"
  fi

  admin_hash="$("$HASH_TOOL" --password "$admin_password")"
  ops_hash="$("$HASH_TOOL" --password "$ops_password")"

  admin_hash_escaped="$(escape_sed_replacement "$admin_hash")"
  ops_hash_escaped="$(escape_sed_replacement "$ops_hash")"
  bearer_token_escaped="$(escape_sed_replacement "$bearer_token")"

  sed -i \
    -e "s/${ADMIN_HASH_PLACEHOLDER}/${admin_hash_escaped}/g" \
    -e "s/${OPS_HASH_PLACEHOLDER}/${ops_hash_escaped}/g" \
    -e "s/${API_TOKEN_PLACEHOLDER}/${bearer_token_escaped}/g" \
    "$CONFIG_FILE"

  echo "Initialized admin and ops passwords in $CONFIG_FILE"
  echo "Generated API bearer token: $bearer_token"
}

generate_tls_certificates() {
  local subject cert_ip
  local tmp_dir ext_file csr_file openssl_cfg mode

  require_openssl

  subject="/O=${TLS_CERT_O}/CN=${TLS_CERT_CN}"
  cert_ip="$(resolve_tls_cert_ip)"
  tmp_dir="$(mktemp -d)"
  ext_file="${tmp_dir}/server.ext"
  csr_file="${tmp_dir}/server.csr"
  openssl_cfg="${tmp_dir}/openssl.cnf"

  cat >"$openssl_cfg" <<EOF
[ req ]
distinguished_name = req_distinguished_name
prompt = no

[ req_distinguished_name ]

[ v3_req ]
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=IP:${cert_ip}
subjectKeyIdentifier=hash
authorityKeyIdentifier=keyid,issuer
EOF

  cat >"$ext_file" <<EOF
[ v3_req ]
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=IP:${cert_ip}
subjectKeyIdentifier=hash
authorityKeyIdentifier=keyid,issuer
EOF

  openssl genrsa -out "$TLS_KEY" 2048 >/dev/null 2>&1
  chmod 600 "$TLS_KEY"

  if [[ -f "$CA_CERT" && -f "$CA_KEY" ]]; then
    mode="CA-signed"
    openssl req -new -key "$TLS_KEY" -out "$csr_file" -subj "$subject" -config "$openssl_cfg" >/dev/null 2>&1
    openssl x509 -req \
      -in "$csr_file" \
      -CA "$CA_CERT" \
      -CAkey "$CA_KEY" \
      -CAserial "${tmp_dir}/myCA.srl" \
      -CAcreateserial \
      -out "$TLS_CERT" \
      -days 825 \
      -sha256 \
      -extfile "$ext_file" \
      -extensions v3_req >/dev/null 2>&1
  else
    mode="self-signed"
    if [[ -f "$CA_CERT" || -f "$CA_KEY" ]]; then
      echo "CA files are incomplete in $CA_DIR. Falling back to a self-signed certificate."
    fi
    openssl req -x509 -new -nodes \
      -key "$TLS_KEY" \
      -out "$TLS_CERT" \
      -days 825 \
      -sha256 \
      -subj "$subject" \
      -config "$openssl_cfg" \
      -extensions v3_req >/dev/null 2>&1
  fi

  chmod 644 "$TLS_CERT"
  rm -rf "$tmp_dir"
  echo "Generated ${mode} TLS certificate for IP ${cert_ip}: $TLS_CERT"
}

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

if [[ -f "$TARGET_DIR/compose.sh" ]]; then
  chmod +x "$TARGET_DIR/compose.sh"
fi

if [[ -f "$TARGET_DIR/load_images.sh" ]]; then
  chmod +x "$TARGET_DIR/load_images.sh"
fi

if [[ -d "$TARGET_DIR/scripts" ]]; then
  find "$TARGET_DIR/scripts" -maxdepth 1 -type f -name "*.sh" -exec chmod +x {} \;
fi

if [[ ! -f "$CONFIG_FILE" && -f "$TARGET_DIR/config.yaml.example" ]]; then
  cp "$TARGET_DIR/config.yaml.example" "$CONFIG_FILE"
  echo "Created $CONFIG_FILE from config.yaml.example"
fi

ensure_initial_secrets
generate_tls_certificates

echo "Install completed."
echo "Next:"
echo "  cd $TARGET_DIR"
echo "  Review and edit config.yaml as needed."
echo "  If you need different host ports, edit docker-compose.yaml before starting."
echo "  In rootless Docker environments, avoid host ports below 1024."
echo "  ./load_images.sh"
echo "  ./compose.sh start"
