#!/bin/bash

HOST="$1"
IP="${2:-127.0.0.1}"  # Usa 127.0.0.1 si no se pasa IP

if [ -z "$HOST" ]; then
    echo "Uso: $0 <host> [ip]"
    exit 1
fi

# Verifica si ya existe
if grep -q -E "\s$HOST(\s|$)" /etc/hosts; then
    echo "El host '$HOST' ya existe en /etc/hosts."
else
    echo "Agregando '$HOST' con IP '$IP' a /etc/hosts..."
    echo "$IP    $HOST" | sudo tee -a /etc/hosts > /dev/null
    echo "Host agregado correctamente."
fi
