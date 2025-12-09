#!/bin/bash

DOMAIN="local.bankfilegen.test"
CERTS_DIR="./certs"
CA_DIR="$CERTS_DIR/ca"
SITE_DIR="$CERTS_DIR/site"

echo "📁 Creando estructura de carpetas..."
mkdir -p "$CA_DIR" "$SITE_DIR"

echo "🔐 Generando clave privada para la CA..."
openssl genrsa -out "$CA_DIR/ca.key.pem" 4096

echo "📄 Generando certificado raíz de la CA..."
openssl req -x509 -new -nodes \
  -key "$CA_DIR/ca.key.pem" \
  -sha256 -days 3650 \
  -out "$CA_DIR/ca.crt.pem" \
  -subj "/C=AR/ST=BuenosAires/L=CABA/O=MiCA Dev/OU=Local CA/CN=Mi Local CA"

echo "🔐 Generando clave privada del sitio..."
openssl genrsa -out "$SITE_DIR/$DOMAIN.key.pem" 2048

echo "📝 Creando archivo CNF para SAN..."
cat > "$SITE_DIR/$DOMAIN.cnf" <<EOF
[req]
default_bits       = 2048
prompt             = no
default_md         = sha256
distinguished_name = dn
req_extensions     = req_ext

[dn]
C = AR
ST = BuenosAires
L = CABA
O = Dev Org
OU = Dev
CN = $DOMAIN

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = $DOMAIN
EOF

echo "📄 Generando CSR..."
openssl req -new \
  -key "$SITE_DIR/$DOMAIN.key.pem" \
  -out "$SITE_DIR/$DOMAIN.csr.pem" \
  -config "$SITE_DIR/$DOMAIN.cnf"

echo "🔏 Firmando CSR con la CA..."
openssl x509 -req \
  -in "$SITE_DIR/$DOMAIN.csr.pem" \
  -CA "$CA_DIR/ca.crt.pem" \
  -CAkey "$CA_DIR/ca.key.pem" \
  -CAcreateserial \
  -out "$SITE_DIR/$DOMAIN.crt.pem" \
  -days 365 \
  -sha256 \
  -extfile "$SITE_DIR/$DOMAIN.cnf" \
  -extensions req_ext

echo "📦 Copiando archivos finales para Traefik..."
cp "$SITE_DIR/$DOMAIN.crt.pem" "$CERTS_DIR/cert.pem"
cp "$SITE_DIR/$DOMAIN.key.pem" "$CERTS_DIR/key.pem"

echo ""
echo "✅ Certificados generados con éxito."
echo "📌 Ahora debes agregar la CA a tu sistema para evitar advertencias:"
echo ""

if [[ "$OSTYPE" == "darwin"* ]]; then
  echo "🔐 macOS:"
  echo "sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain $CA_DIR/ca.crt.pem"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
  echo "🔐 Linux:"
  echo "sudo cp $CA_DIR/ca.crt.pem /usr/local/share/ca-certificates/mi-local-ca.crt"
  echo "sudo update-ca-certificates"
else
  echo "⚠️ Sistema operativo no reconocido. Agregá el certificado raíz manualmente:"
  echo "$CA_DIR/ca.crt.pem"
fi

echo ""
echo "📌 Reiniciá Traefik y tu navegador para aplicar los cambios."
