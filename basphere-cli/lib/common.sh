#!/bin/bash
#
# Basphere CLI 공통 함수 라이브러리
#

# 경로 상수
readonly BASPHERE_CONFIG="/etc/basphere/config.yaml"
readonly BASPHERE_SPECS="/etc/basphere/specs.yaml"
readonly BASPHERE_VSPHERE_ENV="/etc/basphere/vsphere.env"
readonly BASPHERE_DATA_DIR="/var/lib/basphere"
readonly BASPHERE_LOG_DIR="/var/log/basphere"
readonly BASPHERE_LIB_DIR="/usr/local/lib/basphere"

# 색상 정의
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m' # No Color

# ============================================
# 로깅 함수
# ============================================

log_debug() {
    if [[ "${LOG_LEVEL:-info}" == "debug" ]]; then
        echo -e "${CYAN}[DEBUG]${NC} $1" >&2
    fi
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# 감사 로그 기록
audit_log() {
    local action="$1"
    local resource="$2"
    local details="${3:-}"
    local user="${SUDO_USER:-${USER:-unknown}}"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    local log_file="$BASPHERE_LOG_DIR/audit.log"

    echo "$timestamp|$user|$action|$resource|$details" >> "$log_file" 2>/dev/null || true
}

# ============================================
# 설정 파일 로드 함수
# ============================================

# YAML 값 읽기 (yq 사용)
get_config() {
    local key="$1"
    local default="${2:-}"

    if [[ ! -f "$BASPHERE_CONFIG" ]]; then
        echo "$default"
        return
    fi

    local value
    value=$(yq eval "$key // \"\"" "$BASPHERE_CONFIG" 2>/dev/null)

    if [[ -z "$value" || "$value" == "null" ]]; then
        echo "$default"
    else
        echo "$value"
    fi
}

# 스펙 파일에서 값 읽기
get_spec() {
    local key="$1"
    local default="${2:-}"

    if [[ ! -f "$BASPHERE_SPECS" ]]; then
        echo "$default"
        return
    fi

    local value
    value=$(yq eval "$key // \"\"" "$BASPHERE_SPECS" 2>/dev/null)

    if [[ -z "$value" || "$value" == "null" ]]; then
        echo "$default"
    else
        echo "$value"
    fi
}

# vSphere 환경변수 로드
load_vsphere_env() {
    if [[ -f "$BASPHERE_VSPHERE_ENV" ]]; then
        set -a
        source "$BASPHERE_VSPHERE_ENV"
        set +a
    else
        log_error "vSphere 환경변수 파일을 찾을 수 없습니다: $BASPHERE_VSPHERE_ENV"
        return 1
    fi
}

# ============================================
# 사용자 관련 함수
# ============================================

# 현재 사용자 이름 가져오기 (sudo 실행 시 원래 사용자)
get_current_user() {
    echo "${SUDO_USER:-$USER}"
}

# 사용자 데이터 디렉토리
get_user_data_dir() {
    local user="${1:-$(get_current_user)}"
    echo "$BASPHERE_DATA_DIR/users/$user"
}

# 사용자 존재 확인
user_exists() {
    local user="$1"
    [[ -d "$BASPHERE_DATA_DIR/users/$user" ]]
}

# 사용자 메타데이터 읽기
get_user_metadata() {
    local user="$1"
    local key="$2"
    local metadata_file="$BASPHERE_DATA_DIR/users/$user/metadata.json"

    if [[ -f "$metadata_file" ]]; then
        jq -r ".$key // empty" "$metadata_file" 2>/dev/null
    fi
}

# 사용자 메타데이터 쓰기
set_user_metadata() {
    local user="$1"
    local key="$2"
    local value="$3"
    local metadata_file="$BASPHERE_DATA_DIR/users/$user/metadata.json"

    if [[ ! -f "$metadata_file" ]]; then
        echo "{}" > "$metadata_file"
    fi

    local tmp_file=$(mktemp)
    jq ".$key = \"$value\"" "$metadata_file" > "$tmp_file" && mv "$tmp_file" "$metadata_file"
}

# ============================================
# 검증 함수
# ============================================

# 리소스 이름 검증 (영문 소문자, 숫자, 하이픈만 허용)
validate_resource_name() {
    local name="$1"
    local max_length="${2:-63}"

    if [[ -z "$name" ]]; then
        log_error "이름이 비어있습니다"
        return 1
    fi

    if [[ ${#name} -gt $max_length ]]; then
        log_error "이름이 너무 깁니다 (최대 ${max_length}자)"
        return 1
    fi

    if [[ ! "$name" =~ ^[a-z][a-z0-9-]*[a-z0-9]$ && ! "$name" =~ ^[a-z]$ ]]; then
        log_error "이름은 영문 소문자로 시작하고, 영문 소문자/숫자/하이픈만 포함해야 합니다"
        return 1
    fi

    if [[ "$name" =~ -- ]]; then
        log_error "연속된 하이픈(--)은 허용되지 않습니다"
        return 1
    fi

    return 0
}

# VM 이름 중복 확인
vm_name_exists() {
    local user="$1"
    local vm_name="$2"
    local tf_dir="$BASPHERE_DATA_DIR/terraform/$user/$vm_name"

    [[ -d "$tf_dir" ]]
}

# ============================================
# 대화형 입력 함수
# ============================================

# 텍스트 입력 받기
prompt_input() {
    local prompt="$1"
    local default="${2:-}"
    local result

    if [[ -n "$default" ]]; then
        read -rp "$(echo -e "${CYAN}?${NC} $prompt [$default]: ")" result
        result="${result:-$default}"
    else
        read -rp "$(echo -e "${CYAN}?${NC} $prompt: ")" result
    fi

    echo "$result"
}

# 숫자 입력 받기
prompt_number() {
    local prompt="$1"
    local default="${2:-1}"
    local min="${3:-1}"
    local max="${4:-100}"
    local result

    while true; do
        read -rp "$(echo -e "${CYAN}?${NC} $prompt [$default]: ")" result
        result="${result:-$default}"

        if [[ "$result" =~ ^[0-9]+$ ]] && [[ "$result" -ge "$min" ]] && [[ "$result" -le "$max" ]]; then
            echo "$result"
            return 0
        else
            log_error "숫자를 입력하세요 ($min ~ $max)"
        fi
    done
}

# 선택 메뉴
prompt_select() {
    local prompt="$1"
    shift
    local options=("$@")
    local count=${#options[@]}

    # 메뉴는 stderr로 출력 (캡처되지 않도록)
    echo -e "${CYAN}?${NC} $prompt" >&2
    for i in "${!options[@]}"; do
        echo "  [$((i+1))] ${options[$i]}" >&2
    done

    local selection
    while true; do
        read -rp "$(echo -e "${CYAN}?${NC} 선택 [1-$count]: ")" selection

        if [[ "$selection" =~ ^[0-9]+$ ]] && [[ "$selection" -ge 1 ]] && [[ "$selection" -le "$count" ]]; then
            echo "$((selection-1))"  # 0-based index 반환 (stdout)
            return 0
        else
            log_error "1에서 $count 사이의 숫자를 입력하세요"
        fi
    done
}

# 확인 프롬프트
prompt_confirm() {
    local prompt="$1"
    local default="${2:-n}"
    local result

    if [[ "$default" == "y" ]]; then
        read -rp "$(echo -e "${CYAN}?${NC} $prompt [Y/n]: ")" result
        result="${result:-y}"
    else
        read -rp "$(echo -e "${CYAN}?${NC} $prompt [y/N]: ")" result
        result="${result:-n}"
    fi

    [[ "$result" =~ ^[Yy]$ ]]
}

# ============================================
# 파일 락 함수
# ============================================

# 락 획득
acquire_lock() {
    local lock_file="$1"
    local timeout="${2:-30}"

    exec 200>"$lock_file"

    if ! flock -w "$timeout" 200; then
        log_error "락 획득 실패: $lock_file"
        return 1
    fi

    return 0
}

# 락 해제
release_lock() {
    exec 200>&-
}

# ============================================
# 유틸리티 함수
# ============================================

# 타임스탬프 생성
get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# IP 주소 유효성 검사
validate_ip() {
    local ip="$1"
    local ip_regex='^([0-9]{1,3}\.){3}[0-9]{1,3}$'

    if [[ ! "$ip" =~ $ip_regex ]]; then
        return 1
    fi

    IFS='.' read -ra octets <<< "$ip"
    for octet in "${octets[@]}"; do
        if [[ "$octet" -gt 255 ]]; then
            return 1
        fi
    done

    return 0
}

# IP를 정수로 변환
ip_to_int() {
    local ip="$1"
    IFS='.' read -ra octets <<< "$ip"
    echo $(( (${octets[0]} << 24) + (${octets[1]} << 16) + (${octets[2]} << 8) + ${octets[3]} ))
}

# 정수를 IP로 변환
int_to_ip() {
    local int="$1"
    echo "$(( (int >> 24) & 255 )).$(( (int >> 16) & 255 )).$(( (int >> 8) & 255 )).$(( int & 255 ))"
}

# IP 블록 범위 포맷 (시작IP - 끝IP)
format_ip_block_range() {
    local block_start="$1"
    local block_size="${2:-32}"  # 기본값: /27 = 32개

    if [[ -z "$block_start" || "$block_start" == "-" || "$block_start" == "null" ]]; then
        echo "-"
        return
    fi

    local start_int end_int block_end
    start_int=$(ip_to_int "$block_start")
    end_int=$((start_int + block_size - 1))
    block_end=$(int_to_ip "$end_int")

    echo "${block_start} - ${block_end}"
}

# 테이블 형식 출력
print_table_header() {
    local format="$1"
    shift
    printf "${CYAN}$format${NC}\n" "$@"
    printf '%*s\n' "${COLUMNS:-80}" '' | tr ' ' '-'
}

print_table_row() {
    local format="$1"
    shift
    printf "$format\n" "$@"
}

# 스피너 표시
show_spinner() {
    local pid="$1"
    local message="${2:-처리 중...}"
    local spinstr='|/-\'

    while kill -0 "$pid" 2>/dev/null; do
        local temp=${spinstr#?}
        printf "\r${CYAN}%c${NC} %s" "$spinstr" "$message"
        spinstr=$temp${spinstr%"$temp"}
        sleep 0.1
    done
    printf "\r"
}

# ============================================
# API 호출 함수
# ============================================

# API 서버 URL (설정에서 읽거나 기본값 사용)
get_api_url() {
    local url
    url=$(get_config '.api.url' 'http://localhost:8080')
    echo "$url"
}

# API 호출 공통 함수
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    local api_url
    api_url=$(get_api_url)
    local user
    user=$(get_current_user)

    local curl_args=(
        -s
        -X "$method"
        -H "Content-Type: application/json"
        -H "X-Basphere-User: $user"
    )

    if [[ -n "$data" ]]; then
        curl_args+=(-d "$data")
    fi

    curl "${curl_args[@]}" "${api_url}${endpoint}"
}

# API 응답 파싱 함수
api_check_success() {
    local response="$1"
    echo "$response" | jq -r '.success // false'
}

api_get_message() {
    local response="$1"
    echo "$response" | jq -r '.message // ""'
}

api_get_error() {
    local response="$1"
    echo "$response" | jq -r '.errors[]? // .message // "Unknown error"'
}

# API 서버 연결 확인
check_api_connection() {
    local api_url
    api_url=$(get_api_url)

    if ! curl -s --connect-timeout 2 "${api_url}/health" > /dev/null 2>&1; then
        log_error "API 서버에 연결할 수 없습니다: $api_url"
        log_info "API 서버가 실행 중인지 확인하세요."
        return 1
    fi
    return 0
}
