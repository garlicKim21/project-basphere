#!/bin/bash
#
# IPAM 공통 함수
#

# IPAM 디렉토리
readonly IPAM_DIR="/var/lib/basphere/ipam"
readonly ALLOCATIONS_FILE="$IPAM_DIR/allocations.tsv"
readonly LEASES_FILE="$IPAM_DIR/leases.tsv"
readonly IPAM_LOCK="$IPAM_DIR/.lock"

# 공통 라이브러리 로드
source /usr/local/lib/basphere/common.sh 2>/dev/null || source "$(dirname "$0")/../../lib/common.sh"

# 네트워크 설정 로드
load_network_config() {
    NETWORK_CIDR=$(get_config '.network.cidr' '10.254.0.0/21')
    NETWORK_GATEWAY=$(get_config '.network.gateway' '10.254.0.1')
    NETWORK_BLOCK_SIZE=$(get_config '.network.block_size' '32')

    # CIDR에서 시작 IP와 끝 IP 계산
    local base_ip="${NETWORK_CIDR%/*}"
    local prefix="${NETWORK_CIDR#*/}"

    NETWORK_START_INT=$(ip_to_int "$base_ip")
    local host_bits=$((32 - prefix))
    local num_hosts=$((1 << host_bits))
    NETWORK_END_INT=$((NETWORK_START_INT + num_hosts - 1))

    # 예약된 IP 로드
    RESERVED_IPS=()
    while IFS= read -r ip; do
        if [[ -n "$ip" && "$ip" != "null" ]]; then
            RESERVED_IPS+=("$ip")
        fi
    done < <(get_config '.network.reserved[]' '')

    # Gateway는 항상 예약
    if [[ ! " ${RESERVED_IPS[*]} " =~ " ${NETWORK_GATEWAY} " ]]; then
        RESERVED_IPS+=("$NETWORK_GATEWAY")
    fi
}

# IP가 예약되었는지 확인
is_reserved_ip() {
    local ip="$1"
    for reserved in "${RESERVED_IPS[@]}"; do
        if [[ "$ip" == "$reserved" ]]; then
            return 0
        fi
    done
    return 1
}

# 사용자의 할당된 블록 가져오기
get_user_block() {
    local user="$1"

    if [[ ! -f "$ALLOCATIONS_FILE" ]]; then
        return 1
    fi

    local block_start
    block_start=$(grep -v '^#' "$ALLOCATIONS_FILE" 2>/dev/null | awk -F'\t' -v u="$user" '$1 == u {print $2}' || true)

    if [[ -n "$block_start" ]]; then
        echo "$block_start"
        return 0
    fi

    return 1
}

# 다음 사용 가능한 블록 찾기
find_next_available_block() {
    load_network_config

    # 현재 할당된 모든 블록 시작 IP 수집
    local allocated_blocks=()
    if [[ -f "$ALLOCATIONS_FILE" ]]; then
        while IFS=$'\t' read -r user block_start _; do
            [[ "$user" =~ ^#.*$ || -z "$user" ]] && continue
            allocated_blocks+=("$block_start")
        done < "$ALLOCATIONS_FILE"
    fi

    # 첫 번째 블록부터 검색 (예약된 IP 이후)
    local block_start_int=$((NETWORK_START_INT + NETWORK_BLOCK_SIZE))  # 첫 블록 건너뛰기 (예약용)

    while [[ $block_start_int -lt $NETWORK_END_INT ]]; do
        local block_start_ip
        block_start_ip=$(int_to_ip $block_start_int)

        # 이미 할당된 블록인지 확인
        local is_allocated=false
        for allocated in "${allocated_blocks[@]}"; do
            if [[ "$allocated" == "$block_start_ip" ]]; then
                is_allocated=true
                break
            fi
        done

        if [[ "$is_allocated" == "false" ]]; then
            echo "$block_start_ip"
            return 0
        fi

        block_start_int=$((block_start_int + NETWORK_BLOCK_SIZE))
    done

    return 1  # 사용 가능한 블록 없음
}

# 블록 내에서 다음 사용 가능한 IP 찾기
find_next_available_ip() {
    local user="$1"
    local block_start="$2"

    load_network_config

    local block_start_int
    block_start_int=$(ip_to_int "$block_start")
    local block_end_int=$((block_start_int + NETWORK_BLOCK_SIZE - 1))

    # 현재 사용 중인 IP 수집
    local used_ips=()
    if [[ -f "$LEASES_FILE" ]]; then
        while IFS=$'\t' read -r ip lease_user _; do
            [[ "$ip" =~ ^#.*$ || -z "$ip" ]] && continue
            if [[ "$lease_user" == "$user" ]]; then
                used_ips+=("$ip")
            fi
        done < "$LEASES_FILE"
    fi

    # 블록 내에서 첫 번째 사용 가능한 IP 찾기
    local current_int=$block_start_int

    while [[ $current_int -le $block_end_int ]]; do
        local current_ip
        current_ip=$(int_to_ip $current_int)

        # 예약된 IP인지 확인
        if is_reserved_ip "$current_ip"; then
            ((current_int++))
            continue
        fi

        # 이미 사용 중인 IP인지 확인
        local is_used=false
        for used in "${used_ips[@]}"; do
            if [[ "$used" == "$current_ip" ]]; then
                is_used=true
                break
            fi
        done

        if [[ "$is_used" == "false" ]]; then
            echo "$current_ip"
            return 0
        fi

        ((current_int++))
    done

    return 1  # 사용 가능한 IP 없음
}

# 사용자의 IP 사용량 조회
get_user_ip_usage() {
    local user="$1"

    if [[ ! -f "$LEASES_FILE" ]]; then
        echo "0"
        return
    fi

    (grep -v '^#' "$LEASES_FILE" 2>/dev/null || true) | awk -F'\t' -v u="$user" '$2 == u {count++} END {print count+0}'
}

# IP가 사용자 소유인지 확인
is_user_ip() {
    local user="$1"
    local ip="$2"

    if [[ ! -f "$LEASES_FILE" ]]; then
        return 1
    fi

    (grep -v '^#' "$LEASES_FILE" 2>/dev/null || true) | awk -F'\t' -v u="$user" -v i="$ip" '$2 == u && $1 == i {found=1} END {exit !found}'
}
