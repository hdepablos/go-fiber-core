#!/bin/bash

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Ejecutando tests unitarios...${NC}"
echo ""

# Guardar output en archivo temporal
OUTPUT_FILE=$(mktemp)

# Ejecutar tests y procesar línea por línea
go test -v ./... 2>&1 | tee "$OUTPUT_FILE" | while IFS= read -r line; do
    if [[ $line == *"=== RUN"* ]]; then
        TEST_NAME=$(echo "$line" | awk '{print $3}')
        echo -e "${BLUE}🔬 Ejecutando: ${NC}$TEST_NAME"
        
    elif [[ $line == *"--- PASS:"* ]]; then
        TEST_NAME=$(echo "$line" | awk '{print $3}')
        TIME=$(echo "$line" | awk '{print $4}')
        echo -e "${GREEN}  ✓ PASS${NC} $TEST_NAME ${YELLOW}$TIME${NC}"
        
    elif [[ $line == *"--- FAIL:"* ]]; then
        TEST_NAME=$(echo "$line" | awk '{print $3}')
        TIME=$(echo "$line" | awk '{print $4}')
        echo -e "${RED}  ✗ FAIL${NC} $TEST_NAME ${YELLOW}$TIME${NC}"
        
    elif [[ $line == *"FAIL"* ]] && [[ $line == *"coverage:"* ]]; then
        echo -e "${RED}❌ $line${NC}"
        
    elif [[ $line == *"PASS"* ]] && [[ $line == *"coverage:"* ]]; then
        echo -e "${GREEN}✅ $line${NC}"
        
    elif [[ $line == *"ok "* ]]; then
        echo -e "${GREEN}$line${NC}"
        
    elif [[ $line == *"Error Trace:"* ]] || [[ $line == *"Error:"* ]]; then
        echo -e "${RED}    ⚠️  $line${NC}"
        
    elif [[ $line == *"Test:"* ]]; then
        echo -e "${CYAN}    📝 $line${NC}"
        
    elif [[ $line == *"?"* ]] && [[ $line == *"[no test files]"* ]]; then
        :
    else
        if [[ ! -z "$line" ]]; then
            echo "$line"
        fi
    fi
done

EXIT_CODE=${PIPESTATUS[0]}

# Contar solo tests de nivel superior (sin slash en el nombre)
TOTAL=$(grep "^=== RUN" "$OUTPUT_FILE" | grep -v "/" | wc -l)
PASSED=$(grep "^--- PASS:" "$OUTPUT_FILE" | grep -v "/" | wc -l)
FAILED=$(grep "^--- FAIL:" "$OUTPUT_FILE" | grep -v "/" | wc -l)

# Contar subtests
SUBTESTS=$(grep "^=== RUN" "$OUTPUT_FILE" | grep "/" | wc -l)
SUBTESTS_PASSED=$(grep "^--- PASS:" "$OUTPUT_FILE" | grep "/" | wc -l)
SUBTESTS_FAILED=$(grep "^--- FAIL:" "$OUTPUT_FILE" | grep "/" | wc -l)

rm "$OUTPUT_FILE"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BLUE}📊 Resumen de Tests:${NC}"
echo -e "   ${CYAN}Tests principales:${NC}"
echo -e "     Total:   $TOTAL"
echo -e "     ${GREEN}Pasados: $PASSED${NC}"
echo -e "     ${RED}Fallidos: $FAILED${NC}"
echo ""
echo -e "   ${CYAN}Subtests:${NC}"
echo -e "     Total:   $SUBTESTS"
echo -e "     ${GREEN}Pasados: $SUBTESTS_PASSED${NC}"
echo -e "     ${RED}Fallidos: $SUBTESTS_FAILED${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ ¡Todos los tests pasaron exitosamente!${NC}"
else
    echo -e "${RED}❌ Algunos tests fallaron. Revisa los detalles arriba.${NC}"
fi

exit $EXIT_CODE