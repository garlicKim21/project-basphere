#!/bin/bash
#
# Basphere CLI 클러스터 공통 함수 라이브러리
# Stage 2: Cluster API 기반 Kubernetes 클러스터 프로비저닝
#

# 의존성: common.sh가 먼저 로드되어 있어야 함
if [[ -z "${BASPHERE_CONFIG:-}" ]]; then
    echo "Error: common.sh must be sourced before cluster-common.sh" >&2
    exit 1
fi

# ============================================
# 클러스터 관련 경로 상수
# ============================================

readonly BASPHERE_CLUSTERS_DIR="$BASPHERE_DATA_DIR/clusters"
readonly BASPHERE_MGMT_KUBECONFIG="/etc/basphere/management-kubeconfig"
readonly BASPHERE_CAPI_TEMPLATES="/usr/local/lib/basphere/templates/capi"

# ============================================
# Management 클러스터 관련 함수
# ============================================

# Management 클러스터 kubeconfig 경로
get_management_kubeconfig() {
    local kubeconfig
    kubeconfig=$(get_config '.management_cluster.kubeconfig' "$BASPHERE_MGMT_KUBECONFIG")
    echo "$kubeconfig"
}

# Management 클러스터 네임스페이스 프리픽스
get_namespace_prefix() {
    local prefix
    prefix=$(get_config '.management_cluster.namespace_prefix' 'user-')
    echo "$prefix"
}

# 사용자 네임스페이스 이름
get_user_namespace() {
    local user="${1:-$(get_current_user)}"
    local prefix
    prefix=$(get_namespace_prefix)
    echo "${prefix}${user}"
}

# Management 클러스터 연결 확인
check_management_cluster() {
    local kubeconfig
    kubeconfig=$(get_management_kubeconfig)

    if [[ ! -f "$kubeconfig" ]]; then
        log_error "Management 클러스터 kubeconfig를 찾을 수 없습니다: $kubeconfig"
        return 1
    fi

    if ! kubectl --kubeconfig="$kubeconfig" cluster-info &>/dev/null; then
        log_error "Management 클러스터에 연결할 수 없습니다"
        return 1
    fi

    return 0
}

# Management 클러스터에서 kubectl 실행
mgmt_kubectl() {
    local kubeconfig
    kubeconfig=$(get_management_kubeconfig)
    kubectl --kubeconfig="$kubeconfig" "$@"
}

# CAPI가 설치되어 있는지 확인
check_capi_installed() {
    if ! mgmt_kubectl get crd clusters.cluster.x-k8s.io &>/dev/null; then
        log_error "Cluster API가 설치되어 있지 않습니다"
        return 1
    fi
    return 0
}

# CAPV가 설치되어 있는지 확인
check_capv_installed() {
    if ! mgmt_kubectl get crd vsphereclusters.infrastructure.cluster.x-k8s.io &>/dev/null; then
        log_error "CAPV (Cluster API Provider vSphere)가 설치되어 있지 않습니다"
        return 1
    fi
    return 0
}

# ============================================
# 클러스터 스펙 관련 함수
# ============================================

# 클러스터 타입 목록
get_cluster_types() {
    yq eval '.cluster_types | keys | .[]' "$BASPHERE_SPECS" 2>/dev/null
}

# 클러스터 타입 설명
get_cluster_type_description() {
    local type="$1"
    get_spec ".cluster_types.${type}.description" "$type"
}

# 클러스터 타입별 Control Plane 수
get_cluster_control_plane_count() {
    local type="$1"
    get_spec ".cluster_types.${type}.control_plane_count" "1"
}

# 클러스터 타입별 Worker 수
get_cluster_worker_count() {
    local type="$1"
    get_spec ".cluster_types.${type}.worker_count" "2"
}

# 클러스터 노드 스펙 CPU
get_cluster_node_cpu() {
    local spec="$1"
    get_spec ".cluster_node_specs.${spec}.cpu" "2"
}

# 클러스터 노드 스펙 메모리 (MB)
get_cluster_node_memory() {
    local spec="$1"
    get_spec ".cluster_node_specs.${spec}.memory_mb" "4096"
}

# 클러스터 노드 스펙 디스크 (GB)
get_cluster_node_disk() {
    local spec="$1"
    get_spec ".cluster_node_specs.${spec}.disk_gb" "50"
}

# Kubernetes 템플릿 이름
get_kubernetes_template() {
    get_config '.templates.kubernetes' 'ubuntu-2204-kube-v1.28.0'
}

# ============================================
# 클러스터 할당량 관련 함수
# ============================================

# 사용자별 최대 클러스터 수
get_max_clusters() {
    local user="$1"
    local user_max
    user_max=$(get_user_metadata "$user" "max_clusters")

    if [[ -z "$user_max" ]]; then
        get_spec '.cluster_quotas.default.max_clusters' '3'
    else
        echo "$user_max"
    fi
}

# 클러스터당 최대 노드 수
get_max_nodes_per_cluster() {
    get_spec '.cluster_quotas.default.max_nodes_per_cluster' '10'
}

# 사용자의 현재 클러스터 수
get_user_cluster_count() {
    local user="$1"
    local user_cluster_dir="$BASPHERE_CLUSTERS_DIR/$user"

    if [[ ! -d "$user_cluster_dir" ]]; then
        echo "0"
        return
    fi

    find "$user_cluster_dir" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' '
}

# 클러스터 할당량 확인
check_cluster_quota() {
    local user="$1"
    local max_clusters
    local used_clusters

    max_clusters=$(get_max_clusters "$user")
    used_clusters=$(get_user_cluster_count "$user")

    if [[ "$used_clusters" -ge "$max_clusters" ]]; then
        log_error "클러스터 할당량 초과: $used_clusters / $max_clusters"
        return 1
    fi

    return 0
}

# ============================================
# 클러스터 데이터 관련 함수
# ============================================

# 클러스터 데이터 디렉토리
get_cluster_dir() {
    local user="$1"
    local cluster_name="$2"
    echo "$BASPHERE_CLUSTERS_DIR/$user/$cluster_name"
}

# 클러스터 존재 여부 확인
cluster_exists() {
    local user="$1"
    local cluster_name="$2"
    local cluster_dir

    cluster_dir=$(get_cluster_dir "$user" "$cluster_name")
    [[ -d "$cluster_dir" ]]
}

# 클러스터 메타데이터 읽기
get_cluster_metadata() {
    local user="$1"
    local cluster_name="$2"
    local key="$3"
    local metadata_file

    metadata_file="$(get_cluster_dir "$user" "$cluster_name")/metadata.json"

    if [[ -f "$metadata_file" ]]; then
        jq -r ".$key // empty" "$metadata_file" 2>/dev/null
    fi
}

# 클러스터 메타데이터 쓰기
set_cluster_metadata() {
    local user="$1"
    local cluster_name="$2"
    local key="$3"
    local value="$4"
    local cluster_dir metadata_file

    cluster_dir=$(get_cluster_dir "$user" "$cluster_name")
    metadata_file="$cluster_dir/metadata.json"

    mkdir -p "$cluster_dir"

    if [[ ! -f "$metadata_file" ]]; then
        echo "{}" > "$metadata_file"
    fi

    local tmp_file
    tmp_file=$(mktemp)
    jq ".$key = \"$value\"" "$metadata_file" > "$tmp_file" && mv "$tmp_file" "$metadata_file"
}

# 클러스터 메타데이터 전체 저장
save_cluster_metadata() {
    local user="$1"
    local cluster_name="$2"
    local metadata="$3"  # JSON string
    local cluster_dir metadata_file

    cluster_dir=$(get_cluster_dir "$user" "$cluster_name")
    metadata_file="$cluster_dir/metadata.json"

    mkdir -p "$cluster_dir"
    echo "$metadata" > "$metadata_file"
}

# 클러스터 상태 가져오기
get_cluster_status() {
    local user="$1"
    local cluster_name="$2"
    local namespace

    namespace=$(get_user_namespace "$user")

    # Management 클러스터에서 상태 조회
    local status
    status=$(mgmt_kubectl get cluster "$cluster_name" -n "$namespace" -o jsonpath='{.status.phase}' 2>/dev/null)

    if [[ -z "$status" ]]; then
        # CAPI 리소스가 없으면 로컬 메타데이터 확인
        status=$(get_cluster_metadata "$user" "$cluster_name" "status")
    fi

    echo "${status:-unknown}"
}

# ============================================
# kubeconfig 관련 함수
# ============================================

# 워크로드 클러스터 kubeconfig 경로
get_cluster_kubeconfig_path() {
    local user="$1"
    local cluster_name="$2"
    echo "$(get_cluster_dir "$user" "$cluster_name")/kubeconfig"
}

# 워크로드 클러스터 kubeconfig 추출
extract_cluster_kubeconfig() {
    local user="$1"
    local cluster_name="$2"
    local namespace kubeconfig_path

    namespace=$(get_user_namespace "$user")
    kubeconfig_path=$(get_cluster_kubeconfig_path "$user" "$cluster_name")

    # clusterctl을 사용하여 kubeconfig 추출
    clusterctl get kubeconfig "$cluster_name" \
        --kubeconfig "$(get_management_kubeconfig)" \
        --namespace "$namespace" > "$kubeconfig_path" 2>/dev/null

    if [[ -s "$kubeconfig_path" ]]; then
        chmod 600 "$kubeconfig_path"
        return 0
    else
        rm -f "$kubeconfig_path"
        return 1
    fi
}

# ============================================
# CAPI manifest 생성 함수
# ============================================

# 클러스터 manifest 생성
generate_cluster_manifest() {
    local user="$1"
    local cluster_name="$2"
    local cluster_type="$3"
    local worker_spec="$4"
    local control_plane_ip="$5"
    local worker_ips="$6"  # 쉼표로 구분된 IP 목록

    local namespace cluster_dir manifest_file
    namespace=$(get_user_namespace "$user")
    cluster_dir=$(get_cluster_dir "$user" "$cluster_name")
    manifest_file="$cluster_dir/cluster.yaml"

    mkdir -p "$cluster_dir"

    # 클러스터 타입에서 노드 수 가져오기
    local cp_count worker_count cp_spec
    cp_count=$(get_cluster_control_plane_count "$cluster_type")
    worker_count=$(get_cluster_worker_count "$cluster_type")
    cp_spec=$(get_spec ".cluster_types.${cluster_type}.control_plane_spec" "medium")

    # 노드 스펙 가져오기
    local cp_cpu cp_memory cp_disk worker_cpu worker_memory worker_disk
    cp_cpu=$(get_cluster_node_cpu "$cp_spec")
    cp_memory=$(get_cluster_node_memory "$cp_spec")
    cp_disk=$(get_cluster_node_disk "$cp_spec")
    worker_cpu=$(get_cluster_node_cpu "$worker_spec")
    worker_memory=$(get_cluster_node_memory "$worker_spec")
    worker_disk=$(get_cluster_node_disk "$worker_spec")

    # vSphere 설정 가져오기
    local vsphere_server vsphere_datacenter vsphere_cluster vsphere_datastore
    local vsphere_network vsphere_folder vsphere_resource_pool k8s_template
    vsphere_server=$(get_config '.vsphere.server')
    vsphere_datacenter=$(get_config '.vsphere.datacenter')
    vsphere_cluster=$(get_config '.vsphere.cluster')
    vsphere_datastore=$(get_config '.vsphere.datastore')
    vsphere_network=$(get_config '.vsphere.network')
    vsphere_folder=$(get_config '.vsphere.folder')
    vsphere_resource_pool=$(get_config '.vsphere.resource_pool' "/${vsphere_datacenter}/host/${vsphere_cluster}/Resources")
    k8s_template=$(get_kubernetes_template)

    # 템플릿 변수 설정
    export CLUSTER_NAME="$cluster_name"
    export NAMESPACE="$namespace"
    export OWNER="$user"
    export CONTROL_PLANE_IP="$control_plane_ip"
    export CONTROL_PLANE_COUNT="$cp_count"
    export CONTROL_PLANE_CPU="$cp_cpu"
    export CONTROL_PLANE_MEMORY="$cp_memory"
    export CONTROL_PLANE_DISK="$cp_disk"
    export WORKER_COUNT="$worker_count"
    export WORKER_CPU="$worker_cpu"
    export WORKER_MEMORY="$worker_memory"
    export WORKER_DISK="$worker_disk"
    export KUBERNETES_VERSION="v1.28.0"
    export VSPHERE_SERVER="$vsphere_server"
    export VSPHERE_DATACENTER="$vsphere_datacenter"
    export VSPHERE_DATASTORE="$vsphere_datastore"
    export VSPHERE_NETWORK="$vsphere_network"
    export VSPHERE_FOLDER="$vsphere_folder"
    export VSPHERE_RESOURCE_POOL="$vsphere_resource_pool"
    export KUBERNETES_TEMPLATE="$k8s_template"

    # 템플릿 렌더링
    if [[ -f "$BASPHERE_CAPI_TEMPLATES/cluster.yaml.tmpl" ]]; then
        envsubst < "$BASPHERE_CAPI_TEMPLATES/cluster.yaml.tmpl" > "$manifest_file"
    else
        log_error "클러스터 템플릿을 찾을 수 없습니다: $BASPHERE_CAPI_TEMPLATES/cluster.yaml.tmpl"
        return 1
    fi

    echo "$manifest_file"
}

# ============================================
# 사용자 네임스페이스 관리
# ============================================

# 사용자 네임스페이스 생성 (없으면)
ensure_user_namespace() {
    local user="$1"
    local namespace

    namespace=$(get_user_namespace "$user")

    if ! mgmt_kubectl get namespace "$namespace" &>/dev/null; then
        log_info "네임스페이스 생성 중: $namespace"
        mgmt_kubectl create namespace "$namespace"
        mgmt_kubectl label namespace "$namespace" "basphere.dev/owner=$user"
    fi
}

# ============================================
# 대화형 입력 함수 (클러스터용)
# ============================================

# 클러스터 이름 입력
prompt_cluster_name() {
    local name
    while true; do
        name=$(prompt_input "클러스터 이름")
        if validate_resource_name "$name" 2>/dev/null; then
            echo "$name"
            return 0
        fi
    done
}

# 클러스터 타입 선택
prompt_cluster_type() {
    local types=()
    local descriptions=()

    while IFS= read -r type; do
        types+=("$type")
        local desc cp_count worker_count
        desc=$(get_cluster_type_description "$type")
        cp_count=$(get_cluster_control_plane_count "$type")
        worker_count=$(get_cluster_worker_count "$type")
        descriptions+=("$type - $desc (CP: ${cp_count}, Workers: ${worker_count})")
    done < <(get_cluster_types)

    if [[ ${#types[@]} -eq 0 ]]; then
        log_error "클러스터 타입이 정의되지 않았습니다"
        return 1
    fi

    local selection
    selection=$(prompt_select "클러스터 타입" "${descriptions[@]}")
    echo "${types[$selection]}"
}

# Worker 노드 스펙 선택
prompt_worker_spec() {
    local specs=("small" "medium" "large")
    local descriptions=()

    for spec in "${specs[@]}"; do
        local cpu mem disk
        cpu=$(get_cluster_node_cpu "$spec")
        mem=$(get_cluster_node_memory "$spec")
        disk=$(get_cluster_node_disk "$spec")
        descriptions+=("$spec - ${cpu} vCPU, $((mem/1024))GB RAM, ${disk}GB Disk")
    done

    local selection
    selection=$(prompt_select "Worker 노드 스펙" "${descriptions[@]}")
    echo "${specs[$selection]}"
}

# ============================================
# 클러스터 목록 출력
# ============================================

# 클러스터 목록 출력
list_user_clusters() {
    local user="$1"
    local user_cluster_dir="$BASPHERE_CLUSTERS_DIR/$user"

    if [[ ! -d "$user_cluster_dir" ]]; then
        return
    fi

    for cluster_dir in "$user_cluster_dir"/*/; do
        if [[ -d "$cluster_dir" ]]; then
            local cluster_name
            cluster_name=$(basename "$cluster_dir")
            local status type cp_ip created_at

            status=$(get_cluster_status "$user" "$cluster_name")
            type=$(get_cluster_metadata "$user" "$cluster_name" "type")
            cp_ip=$(get_cluster_metadata "$user" "$cluster_name" "control_plane_ip")
            created_at=$(get_cluster_metadata "$user" "$cluster_name" "created_at")

            echo "$cluster_name|$type|$status|$cp_ip|$created_at"
        fi
    done
}
