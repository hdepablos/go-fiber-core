#!/bin/bash

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Ejecutando tests unitarios...${NC}"
echo ""

# Simplemente pasar todo el output con colores
go test -v ./... 2>&1 | sed \
    -e "s/^=== RUN.*/${BLUE}&${NC}/" \
    -e "s/^--- PASS:.*/${GREEN}&${NC}/" \
    -e "s/^--- FAIL:.*/${RED}&${NC}/" \
    -e "s/^.*Error Trace:.*/${RED}&${NC}/" \
    -e "s/^.*Error:.*/${RED}&${NC}/" \
    -e "s/^.*Expected.*/${YELLOW}&${NC}/" \
    -e "s/^.*Actual.*/${YELLOW}&${NC}/" \
    -e "s/^PASS.*/${GREEN}&${NC}/" \
    -e "s/^FAIL.*/${RED}&${NC}/" \
    -e "s/panic.*/${RED}&${NC}/"

EXIT_CODE=${PIPESTATUS[0]}

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ ¡Todos los tests pasaron exitosamente!${NC}"
else
    echo -e "${RED}❌ Algunos tests fallaron. Revisa los detalles arriba.${NC}"
fi

exit $EXIT_CODE