#!/usr/bin/env bash
# ============================================================================
# Script optimizado para importación masiva de CSV por lotes
# Versión: 2.1 - Con salida visual mejorada
# ============================================================================

set -euo pipefail

# ============================================================================
# CONFIGURACIÓN INICIAL
# ============================================================================

readonly SCRIPT_VERSION="2.1"
readonly SCRIPT_NAME="$(basename "${BASH_SOURCE[0]}")"
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ============================================================================
# CARGA DE ARCHIVO .ENV
# ============================================================================

load_env_file() {
    local env_file="${1:-}"
    
    # Buscar archivo .env en orden de prioridad
    local env_paths=(
        "${env_file}"                                    # Argumento explícito
        "${SCRIPT_DIR}/.env"                            # Mismo directorio que el script
        "${SCRIPT_DIR}/../.env"                         # Directorio padre
        "$(pwd)/.env"                                   # Directorio actual
        "${HOME}/.config/${SCRIPT_NAME}/.env"           # Configuración de usuario
    )
    
    for env_path in "${env_paths[@]}"; do
        if [[ -n "${env_path}" && -f "${env_path}" && -r "${env_path}" ]]; then
            echo "📁 Loading configuration from: ${env_path}" >&2
            set -a  # Automatically export all variables
            # shellcheck source=/dev/null
            source "${env_path}"
            set +a
            return 0
        fi
    done
    
    echo "⚠️  Warning: No .env file found, using defaults" >&2
    return 1
}

# Cargar configuración (puede pasar un archivo .env específico como argumento)
load_env_file "${ENV_FILE:-}"

# ============================================================================
# CONFIGURACIÓN CON VALORES POR DEFECTO
# ============================================================================

# Archivo a importar (puede pasarse como argumento al script)
FILE="${1:-${FILE:-tests/file-import-all.csv}}"

# Configuración del negocio
BRANCH_ID="${BRANCH_ID:-1}"
REF_CODE="${REF_CODE:-50}"
BATCH="${BATCH:-20000}"
X_CLIENT_CODE="${X_CLIENT_CODE:-cron}"

# Configuración de la API
URL_BASE="${URL_BASE:-http://127.0.0.1:9009/}"
EMAIL="${EMAIL:-hdepablos@libgot.com}"
PASSWORD="${PASSWORD:-123456}"

# Configuración de reintentos y rate limiting
RETRY_MAX="${RETRY_MAX:-8}"
RATE_LIMIT_WINDOW_SECONDS="${RATE_LIMIT_WINDOW_SECONDS:-60}"
IMPORTS_RATE_LIMIT_PER_MINUTE="${IMPORTS_RATE_LIMIT_PER_MINUTE:-5000}"

# Configuración avanzada
CURL_CONNECT_TIMEOUT="${CURL_CONNECT_TIMEOUT:-10}"
CURL_MAX_TIME="${CURL_MAX_TIME:-300}"
LOG_LEVEL="${LOG_LEVEL:-INFO}"
TMP_DIR_BASE="${TMP_DIR_BASE:-/tmp}"

# Constantes calculadas
readonly RATE_LIMIT_WINDOW_SLEEP_SECONDS="$((RATE_LIMIT_WINDOW_SECONDS + 5))"
readonly PER_REQUEST_SLEEP_SECONDS="$(awk "BEGIN {printf \"%.6f\", 60/${IMPORTS_RATE_LIMIT_PER_MINUTE}}")"

# Variables globales (inicializadas)
tmp_dir=""

# ============================================================================
# FUNCIONES UTILITARIAS
# ============================================================================

# Iconos para diferentes tipos de mensajes
get_icon() {
    local level="${1:-INFO}"
    case "${level}" in
        INFO)    echo "ℹ️ " ;;
        SUCCESS) echo "✅ " ;;
        WARN)    echo "⚠️ " ;;
        ERROR)   echo "❌ " ;;
        DEBUG)   echo "🔍 " ;;
        UPLOAD)  echo "📤 " ;;
        SPLIT)   echo "✂️  " ;;
        CLOCK)   echo "⏱️  " ;;
        CONFIG)  echo "⚙️  " ;;
        AUTH)    echo "🔐 " ;;
        RATE)    echo "🚦 " ;;
        *)       echo "   " ;;
    esac
}

# Logging con niveles, colores e iconos
log() {
    local level="${1:-INFO}"
    local message="${2:-}"
    local timestamp="$(date '+%H:%M:%S')"
    local icon="$(get_icon "${level}")"
    
    # Definir niveles de log
    local level_priority=0
    case "${LOG_LEVEL}" in
        DEBUG) level_priority=0 ;;
        INFO)  level_priority=1 ;;
        WARN)  level_priority=2 ;;
        ERROR) level_priority=3 ;;
        *)     level_priority=1 ;;
    esac
    
    local msg_priority=0
    case "${level}" in
        DEBUG) msg_priority=0 ;;
        INFO)  msg_priority=1 ;;
        WARN)  msg_priority=2 ;;
        ERROR) msg_priority=3 ;;
        SUCCESS) msg_priority=1 ;;
        *)     msg_priority=1 ;;
    esac
    
    # Solo mostrar si el nivel es suficientemente alto
    if [[ ${msg_priority} -ge ${level_priority} ]]; then
        echo "${icon}[${timestamp}] ${message}" >&2
    fi
}

# Manejo de errores
error_exit() {
    local error_message="${1:-Unknown error}"
    local exit_code="${2:-1}"
    log "ERROR" "${error_message}"
    log "ERROR" "Exiting with code ${exit_code}"
    
    # Limpiar directorio temporal si existe
    if [[ -n "${tmp_dir}" && -d "${tmp_dir}" ]]; then
        rm -rf "${tmp_dir}" 2>/dev/null || true
        log "DEBUG" "Cleaned up temporary directory"
    fi
    
    exit "${exit_code}"
}

# Validación de dependencias
check_dependencies() {
    local deps=("curl" "awk" "python3")
    local missing=()
    
    for dep in "${deps[@]}"; do
        if ! command -v "${dep}" &> /dev/null; then
            missing+=("${dep}")
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        error_exit "Missing dependencies: ${missing[*]}. Please install them and try again." 2
    fi
    
    log "DEBUG" "All dependencies are available"
}

# Normalizar URL base
normalize_url() {
    local url="${1}"
    echo "${url}" | sed 's:/*$:/:'
}

# Validar archivo de entrada
validate_input_file() {
    local file_path="${1}"
    
    if [[ ! -f "${file_path}" ]]; then
        error_exit "File not found: ${file_path}" 2
    fi
    
    if [[ ! -r "${file_path}" ]]; then
        error_exit "File not readable: ${file_path}" 2
    fi
    
    local line_count
    line_count="$(wc -l < "${file_path}" | tr -d '[:space:]')"
    
    if [[ -z "${line_count}" || "${line_count}" -lt 2 ]]; then
        error_exit "File must contain header + at least 1 data row: ${file_path}" 2
    fi
    
    echo "${line_count}"
}

# Formatear tiempo transcurrido
format_elapsed_time() {
    local start_ns="$1"
    local end_ns="$2"
    local elapsed_ns="$((end_ns - start_ns))"
    local elapsed_ms="$((elapsed_ns / 1000000))"
    
    local hours="$((elapsed_ms / 3600000))"
    local minutes="$(((elapsed_ms % 3600000) / 60000))"
    local seconds="$(((elapsed_ms % 60000) / 1000))"
    local milliseconds="$((elapsed_ms % 1000))"
    
    if [[ ${hours} -gt 0 ]]; then
        printf "%dh %dm %ds" "${hours}" "${minutes}" "${seconds}"
    elif [[ ${minutes} -gt 0 ]]; then
        printf "%dm %ds" "${minutes}" "${seconds}"
    else
        printf "%d.%03ds" "${seconds}" "${milliseconds}"
    fi
}

# Mostrar barra de progreso
show_progress() {
    local current="$1"
    local total="$2"
    local width=30
    local percentage=$((current * 100 / total))
    local filled=$((percentage * width / 100))
    local empty=$((width - filled))
    
    printf "\r%s" "$(get_icon "UPLOAD")Progress: ["
    printf "%${filled}s" | tr ' ' '█'
    printf "%${empty}s" | tr ' ' '░'
    printf "] %3d%% (%d/%d)" "${percentage}" "${current}" "${total}"
}

# ============================================================================
# FUNCIONES DE API
# ============================================================================

# Autenticación con backoff exponencial
authenticate() {
    local login_url="${1}"
    local x_client_code="${2}"
    local email="${3}"
    local password="${4}"
    local attempt=1
    
    log "AUTH" "Authenticating to ${login_url}..."
    
    while [[ ${attempt} -le ${RETRY_MAX} ]]; do
        local response_file
        response_file="$(mktemp)"
        
        log "DEBUG" "Login attempt ${attempt}/${RETRY_MAX}"
        
        local http_code
        http_code="$(
            curl -sS -o "${response_file}" -w "%{http_code}" \
                -X POST "${login_url}" \
                -H "Content-Type: application/json" \
                -H "X-Client-Code: ${x_client_code}" \
                -d "{\"email\":\"${email}\",\"password\":\"${password}\"}" \
                --connect-timeout "${CURL_CONNECT_TIMEOUT}" \
                --max-time "${CURL_MAX_TIME}" \
                2>/dev/null
        )"
        
        if [[ "${http_code}" == "200" ]]; then
            # Extraer token usando jq si está disponible, sino python
            local token
            if command -v jq &> /dev/null; then
                token="$(jq -r '.data.access_token // empty' "${response_file}" 2>/dev/null)"
            else
                token="$(
                    python3 -c "
import json, sys
try:
    with open('${response_file}', 'r') as f:
        data = json.load(f)
    token = data.get('data', {}).get('access_token', '')
    if token:
        print(token)
except:
    pass
" 2>/dev/null
                )"
            fi
            
            rm -f "${response_file}"
            
            if [[ -z "${token}" ]]; then
                error_exit "Token not found in response" 3
            fi
            
            log "SUCCESS" "Authentication successful"
            echo "${token}"
            return 0
        fi
        
        # Rate limit detection
        if [[ "${http_code}" == "429" ]]; then
            local wait_time=$((RATE_LIMIT_WINDOW_SLEEP_SECONDS * attempt))
            log "RATE" "Rate limit hit, waiting ${wait_time}s before retry (${attempt}/${RETRY_MAX})"
            rm -f "${response_file}"
            sleep "${wait_time}"
            attempt=$((attempt + 1))
            continue
        fi
        
        # Error fatal
        log "ERROR" "Login failed with HTTP ${http_code}"
        if [[ -s "${response_file}" ]]; then
            log "ERROR" "Response: $(head -c 500 "${response_file}" | tr '\n' ' ')"
        fi
        rm -f "${response_file}"
        error_exit "Authentication failed" 3
    done
    
    error_exit "Max retries exceeded for authentication" 4
}

# Subida de archivo con backoff exponencial
upload_chunk() {
    local url="${1}"
    local token="${2}"
    local x_client_code="${3}"
    local chunk_file="${4}"
    local attempt=1
    
    local chunk_name
    chunk_name="$(basename "${chunk_file}")"
    
    while [[ ${attempt} -le ${RETRY_MAX} ]]; do
        local response_file
        response_file="$(mktemp)"
        
        local http_code
        http_code="$(
            curl -sS -o "${response_file}" -w "%{http_code}" \
                -X POST "${url}" \
                -H "Authorization: Bearer ${token}" \
                -H "X-Client-Code: ${x_client_code}" \
                -F "file=@${chunk_file}" \
                --connect-timeout "${CURL_CONNECT_TIMEOUT}" \
                --max-time "${CURL_MAX_TIME}" \
                2>/dev/null
        )"
        
        if [[ "${http_code}" == "200" ]]; then
            rm -f "${response_file}"
            return 0
        fi
        
        # Rate limit con backoff exponencial
        if [[ "${http_code}" == "429" ]]; then
            local wait_time=$((RATE_LIMIT_WINDOW_SLEEP_SECONDS * attempt))
            log "RATE" "Rate limit on ${chunk_name}, retry in ${wait_time}s (${attempt}/${RETRY_MAX})"
            rm -f "${response_file}"
            sleep "${wait_time}"
            attempt=$((attempt + 1))
            continue
        fi
        
        # Error recuperable (5xx)
        if [[ "${http_code}" -ge 500 && "${http_code}" -le 599 ]]; then
            local wait_time=$((2 ** attempt))
            log "WARN" "Server error ${http_code} on ${chunk_name}, retry in ${wait_time}s (${attempt}/${RETRY_MAX})"
            rm -f "${response_file}"
            sleep "${wait_time}"
            attempt=$((attempt + 1))
            continue
        fi
        
        # Error fatal
        log "ERROR" "Upload failed for ${chunk_name} (HTTP ${http_code})"
        if [[ -s "${response_file}" ]]; then
            log "ERROR" "Response: $(head -c 500 "${response_file}" | tr '\n' ' ')"
        fi
        rm -f "${response_file}"
        return 1
    done
    
    log "ERROR" "Max retries exceeded for ${chunk_name}"
    return 1
}

# ============================================================================
# DIVISIÓN DE ARCHIVOS
# ============================================================================

split_csv_into_chunks() {
    local input_file="${1}"
    local batch_size="${2}"
    local output_dir="${3}"
    
    log "SPLIT" "Splitting ${input_file} into chunks of ${batch_size} rows..."
    
    # Usar awk para dividir el archivo
    local last_chunk
    last_chunk="$(
        awk -v batch="${batch_size}" -v outdir="${output_dir}" '
        BEGIN {
            header = ""
            row_count = 0
        }
        
        NR == 1 {
            header = $0
            next
        }
        
        {
            row_count++
            chunk_idx = int((row_count - 1) / batch) + 1
            fname = sprintf("%s/chunk_%04d.csv", outdir, chunk_idx)
            
            if (!(chunk_idx in files_opened)) {
                files_opened[chunk_idx] = 1
                print header > fname
            }
            
            print $0 >> fname
        }
        
        END {
            if (row_count > 0) {
                print int((row_count - 1) / batch) + 1
            } else {
                print 0
            }
        }
        ' "${input_file}"
    )"
    
    # Verificar que se generaron chunks
    if [[ -z "${last_chunk}" || "${last_chunk}" -eq 0 ]]; then
        error_exit "No chunks were generated from the input file" 2
    fi
    
    log "SUCCESS" "Generated ${last_chunk} chunks"
    echo "${last_chunk}"
}

# ============================================================================
# FUNCIÓN PRINCIPAL
# ============================================================================

main() {
    # Tiempo de inicio
    local start_time_ns
    start_time_ns="$(date +%s%N)"
    
    # Mostrar banner
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║     📦 CSV Import Tool v${SCRIPT_VERSION} - Bulk Uploader            ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    
    # Mostrar configuración
    log "CONFIG" "Configuration Summary:"
    log "CONFIG" "  📁 Input File     : ${FILE}"
    log "CONFIG" "  🏢 Branch ID      : ${BRANCH_ID}"
    log "CONFIG" "  🔖 Reference Code : ${REF_CODE}"
    log "CONFIG" "  📊 Batch Size     : ${BATCH} rows/chunk"
    log "CONFIG" "  🚦 Rate Limit     : ${IMPORTS_RATE_LIMIT_PER_MINUTE} requests/minute"
    log "CONFIG" "  ⏱️  Sleep/Request  : ${PER_REQUEST_SLEEP_SECONDS} seconds"
    log "CONFIG" "  🔄 Max Retries    : ${RETRY_MAX}"
    log "CONFIG" "  🌐 API Base URL   : ${URL_BASE}"
    echo ""
    
    # Validar dependencias
    check_dependencies
    
    # Normalizar URL base
    local url_base_normalized
    url_base_normalized="$(normalize_url "${URL_BASE}")"
    
    # Construir URLs
    local login_url="${url_base_normalized}api/v1/auth/login"
    local imports_base_url="${url_base_normalized}api/v1/imports/all"
    local logout_url="${url_base_normalized}api/v1/auth/logout"
    
    # Validar archivo de entrada
    local total_lines
    total_lines="$(validate_input_file "${FILE}")"
    local data_rows="$((total_lines - 1))"
    log "INFO" "📄 File analysis: ${total_lines} total lines (${data_rows} data rows)"
    
    # Generar código único para esta ejecución
    local key_code
    key_code="$(date +%y%m%d%H%M%S)$(printf '%04d' $((RANDOM % 9999 + 1000)))"
    log "INFO" "🔑 Execution Key   : ${key_code}"
    echo ""
    
    # Autenticación
    local token
    token="$(authenticate "${login_url}" "${X_CLIENT_CODE}" "${EMAIL}" "${PASSWORD}")"
    echo ""
    
    # Preparar directorio temporal
    tmp_dir="$(mktemp -d -p "${TMP_DIR_BASE}" "import_chunks_XXXXXX")"
    trap 'cleanup' EXIT INT TERM
    log "DEBUG" "Temporary directory: ${tmp_dir}"
    
    # Dividir archivo en chunks
    local chunks_generated
    chunks_generated="$(split_csv_into_chunks "${FILE}" "${BATCH}" "${tmp_dir}")"
    
    # Obtener lista de chunks ordenada
    mapfile -t chunks < <(find "${tmp_dir}" -name "chunk_*.csv" -type f | sort -V)
    local total_chunks="${#chunks[@]}"
    
    if [[ "${total_chunks}" -ne "${chunks_generated}" ]]; then
        log "WARN" "Chunk count mismatch: expected ${chunks_generated}, found ${total_chunks}"
    fi
    
    echo ""
    log "UPLOAD" "Starting upload of ${total_chunks} chunks..."
    echo ""
    
    # Procesar cada chunk
    local success_count=0
    local chunk_index=1
    local failed_chunks=()
    
    for chunk_file in "${chunks[@]}"; do
        # Construir URL para este chunk
        local url="${imports_base_url}/${BRANCH_ID}/${REF_CODE}/${total_lines}/${key_code}"
        local chunk_name="$(basename "${chunk_file}")"
        
        # Mostrar progreso
        show_progress "${chunk_index}" "${total_chunks}"
        
        # Subir chunk
        if upload_chunk "${url}" "${token}" "${X_CLIENT_CODE}" "${chunk_file}"; then
            ((success_count++))
            log "SUCCESS" "✓ ${chunk_name}" >&2
        else
            failed_chunks+=("${chunk_name}")
            log "ERROR" "✗ ${chunk_name} - Upload failed" >&2
        fi
        
        # Respetar rate limit (excepto en el último chunk)
        if [[ "${PER_REQUEST_SLEEP_SECONDS}" != "0.000000" ]] && \
           [[ "${PER_REQUEST_SLEEP_SECONDS}" != "0" ]] && \
           [[ "${chunk_index}" -lt "${total_chunks}" ]]; then
            sleep "${PER_REQUEST_SLEEP_SECONDS}"
        fi
        
        ((chunk_index++))
    done
    
    # Limpiar línea de progreso
    echo ""
    echo ""
    
    # Logout (best effort)
    log "DEBUG" "Performing logout"
    curl -sS -o /dev/null \
        -X POST "${logout_url}" \
        -H "Authorization: Bearer ${token}" \
        -H "X-Client-Code: ${X_CLIENT_CODE}" \
        -H "Content-Type: application/json" \
        -d '{}' \
        --max-time 5 2>/dev/null || true
    
    # Estadísticas finales
    local end_time_ns
    end_time_ns="$(date +%s%N)"
    local elapsed_time
    elapsed_time="$(format_elapsed_time "${start_time_ns}" "${end_time_ns}")"
    
    # Resultado final con formato visual
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    
    if [[ "${success_count}" -eq "${total_chunks}" ]]; then
        echo "  ✅  SUCCESS: All chunks uploaded successfully!              "
        echo "╠══════════════════════════════════════════════════════════════╣"
        echo "  📊 Statistics:                                              "
        printf "     • Total Chunks    : %4d                                      \n" "${success_count}"
        printf "     • Total Lines     : %4d                                      \n" "${total_lines}"
        printf "     • Data Rows       : %4d                                      \n" "${data_rows}"
        printf "     • Execution Key   : %s                      \n" "${key_code}"
        printf "     • Time Elapsed    : %s                                   \n" "${elapsed_time}"
        echo "╠══════════════════════════════════════════════════════════════╣"
        echo "  ⚙️  Configuration:                                            "
        printf "     • Rate Limit      : %4d requests/minute                         \n" "${IMPORTS_RATE_LIMIT_PER_MINUTE}"
        printf "     • Sleep/Request   : %s seconds                                  \n" "${PER_REQUEST_SLEEP_SECONDS}"
        printf "     • Expected Chunks : %4d                                      \n" "${chunks_generated}"
        echo "╚══════════════════════════════════════════════════════════════╝"
        echo ""
        return 0
    else
        echo "  ❌  FAILED: ${success_count}/${total_chunks} chunks uploaded successfully   "
        echo "╠══════════════════════════════════════════════════════════════╣"
        echo "  📊 Statistics:                                              "
        printf "     • Success        : %4d chunks                                 \n" "${success_count}"
        printf "     • Failed         : %4d chunks                                 \n" "$((total_chunks - success_count))"
        printf "     • Total Lines    : %4d                                      \n" "${total_lines}"
        printf "     • Time Elapsed   : %s                                   \n" "${elapsed_time}"
        if [[ ${#failed_chunks[@]} -gt 0 ]]; then
            echo "╠══════════════════════════════════════════════════════════════╣"
            echo "  ❌ Failed Chunks:                                            "
            for failed in "${failed_chunks[@]}"; do
                printf "     • %s\n" "${failed}"
            done
        fi
        echo "╚══════════════════════════════════════════════════════════════╝"
        echo ""
        return 1
    fi
}

# ============================================================================
# FUNCIÓN DE LIMPIEZA
# ============================================================================

cleanup() {
    if [[ -n "${tmp_dir}" && -d "${tmp_dir}" ]]; then
        rm -rf "${tmp_dir}" 2>/dev/null || true
        log "DEBUG" "Cleaned up temporary directory: ${tmp_dir}"
    fi
}

# ============================================================================
# EJECUCIÓN
# ============================================================================

# Capturar señales para limpieza
trap cleanup EXIT INT TERM

# Ejecutar función principal
if ! main "$@"; then
    exit 1
fi