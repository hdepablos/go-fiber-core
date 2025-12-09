#!/bin/bash

# run_tests.sh

echo "🧪  Ejecutando tests..."

# Ejecutamos los tests y guardamos la salida en un archivo temporal.
# El flag -json nos da una salida estructurada que es más fácil de procesar.
# Redirigimos stderr a stdout (2>&1) para capturar todos los errores.
go test ./... -json > test_results.log 2>&1

# Verificamos el código de salida del último comando. 0 significa éxito.
if [ $? -eq 0 ]; then
    echo ""
    echo "✅ ¡Todos los tests pasaron exitosamente!"
    # Opcional: limpiar el log si todo salió bien
    rm test_results.log
else
    echo ""
    echo "❌ ¡Fallaron algunos tests! Aquí está el resumen de errores:"
    echo "-----------------------------------------------------------"

    # Usamos `jq` para parsear el JSON y mostrar solo los tests que fallaron y su output.
    # Esto filtra todo el ruido y te muestra solo lo que necesitas ver.
    cat test_results.log | jq -r 'select(.Action == "fail") | "\n🔴 Test Fallido: \(.Test)\nOutput:\n\(.Output)"'

    # Si no tienes `jq`, puedes usar `grep` como una alternativa más simple:
    # cat test_results.log | grep -E "FAIL|Error:|panic:"

    echo "-----------------------------------------------------------"
    # Salimos con un código de error para que Make también falle.
    exit 1
fi