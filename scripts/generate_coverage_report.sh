#!/bin/bash
# scripts/generate_coverage_report.sh

# Este script asume que ya existe un archivo 'coverage.out' en la raíz.

echo ""
echo "📊 Resumen de Cobertura por Función:"
echo "----------------------------------"
# Mostramos un resumen legible en la terminal, indicando qué funciones
# de tu código están cubiertas por los tests.
go tool cover -func=coverage.out

# Generamos el reporte HTML interactivo.
go tool cover -html=coverage.out -o coverage.html

echo ""
echo "✅ ¡Reporte interactivo generado!"
echo "🔍 Para verlo, abrí el archivo 'coverage.html' en tu navegador."