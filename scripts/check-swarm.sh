#!/bin/bash

# se implemenra así:
# check-swarm:
#	@./scripts/check-swarm.sh

YELLOW='\033[0;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

if [ "$(docker info --format '{{.Swarm.LocalNodeState}}')" != "active" ]; then
    echo -e "${YELLOW}[INFO] Docker Swarm no está activo. Ejecutando 'docker swarm init'...${NC}"
    docker swarm init
else
    echo -e "${GREEN}[INFO] Docker Swarm ya está activo.${NC}"
fi

read -p "¿Deseas salir del modo Swarm? (y/n): " confirm
if [ "$confirm" = "y" ]; then
    docker swarm leave --force
    echo -e "${GREEN}[INFO] Saliste del modo Swarm.${NC}"
else
    echo -e "${YELLOW}[INFO] Acción cancelada.${NC}"
fi
