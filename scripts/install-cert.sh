#!/bin/bash
set -euo pipefail

# Install mitmix CA cert into system trust store
# Usage: ./scripts/install-cert.sh [--hub ws://host:8090]

HUB="${1#--hub=}"
if [ -z "$HUB" ] && [ "$1" = "--hub" ] && [ -n "$2" ]; then
    HUB="$2"
fi

CERT="$HOME/.mitmproxy/mitmproxy-ca-cert.pem"

if [ -n "$HUB" ]; then
    REST="${HUB/ws:/http:}"
    echo "Downloading CA cert from $REST/api/mitm/ca-cert..."
    curl -sf "$REST/api/mitm/ca-cert" -o /tmp/mitmix-ca.pem
    CERT="/tmp/mitmix-ca.pem"
elif [ -f "$CERT" ]; then
    echo "Using local cert at $CERT"
else
    echo "No cert found. Run mitmproxy first, or provide --hub URL."
    exit 1
fi

if [ "$(uname)" = "Darwin" ]; then
    echo "Installing to macOS System keychain (requires sudo)..."
    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "$CERT"
    echo "Done."
elif [ "$(uname)" = "Linux" ]; then
    echo "Installing to /usr/local/share/ca-certificates/..."
    sudo cp "$CERT" /usr/local/share/ca-certificates/mitmix-ca-cert.crt
    sudo update-ca-certificates
    echo "Done."
else
    echo "Unsupported OS. Cert saved at $CERT — install manually."
fi
