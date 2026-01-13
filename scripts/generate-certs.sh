#!/bin/bash
# Generate self-signed certificates for HTTPS

set -e

CERT_DIR="${1:-./certs}"
DOMAIN="${2:-localhost}"

mkdir -p "$CERT_DIR"

# Generate CA key and certificate
openssl genrsa -out "$CERT_DIR/ca.key" 4096
openssl req -new -x509 -days 3650 -key "$CERT_DIR/ca.key" -out "$CERT_DIR/ca.crt" \
    -subj "/C=US/ST=State/L=City/O=Terraform Registry/CN=Terraform Registry CA"

# Generate server key
openssl genrsa -out "$CERT_DIR/server.key" 2048

# Create SAN config
cat > "$CERT_DIR/san.cnf" << EOF
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = req_ext

[dn]
C = US
ST = State
L = City
O = Terraform Registry
CN = $DOMAIN

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = $DOMAIN
DNS.2 = localhost
DNS.3 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Generate server CSR
openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" \
    -config "$CERT_DIR/san.cnf"

# Sign server certificate with CA
openssl x509 -req -days 3650 -in "$CERT_DIR/server.csr" \
    -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial \
    -out "$CERT_DIR/server.crt" \
    -extfile "$CERT_DIR/san.cnf" -extensions req_ext

# Clean up
rm -f "$CERT_DIR/server.csr" "$CERT_DIR/san.cnf" "$CERT_DIR/ca.srl"

echo ""
echo "✅ Certificates generated in $CERT_DIR/"
echo ""
echo "Files created:"
echo "  - ca.crt      (CA certificate - add to system trust store)"
echo "  - ca.key      (CA private key - keep secure)"
echo "  - server.crt  (Server certificate)"
echo "  - server.key  (Server private key)"
echo ""
echo "To trust the CA certificate:"
echo ""
echo "  macOS:"
echo "    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain $CERT_DIR/ca.crt"
echo ""
echo "  Linux:"
echo "    sudo cp $CERT_DIR/ca.crt /usr/local/share/ca-certificates/terraform-registry-ca.crt"
echo "    sudo update-ca-certificates"
echo ""
echo "  Terraform CLI (per-user):"
echo "    export SSL_CERT_FILE=$CERT_DIR/ca.crt"
echo "    # Or in .terraformrc:"
echo "    # provider_installation {"
echo "    #   network_mirror {"
echo "    #     url = \"https://$DOMAIN:3443/\""
echo "    #   }"
echo "    # }"
