#!/bin/bash
#
# Basphere CLI 설치 스크립트
# Bastion 서버에서 초기 설정을 수행합니다.
#
# 사용법: sudo ./install.sh
#

set -euo pipefail

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 로그 함수
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Root 권한 확인
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "이 스크립트는 root 권한으로 실행해야 합니다."
        log_info "사용법: sudo $0"
        exit 1
    fi
}

# 필수 패키지 확인
check_dependencies() {
    log_info "필수 패키지 확인 중..."

    local missing=()

    # jq - JSON 파싱
    if ! command -v jq &> /dev/null; then
        missing+=("jq")
    fi

    # yq - YAML 파싱
    if ! command -v yq &> /dev/null; then
        missing+=("yq")
    fi

    # terraform
    if ! command -v terraform &> /dev/null; then
        missing+=("terraform")
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_warn "다음 패키지가 설치되어 있지 않습니다: ${missing[*]}"
        log_info "설치를 진행합니다..."

        # OS 감지
        if [[ -f /etc/debian_version ]]; then
            apt-get update

            for pkg in "${missing[@]}"; do
                case $pkg in
                    jq)
                        apt-get install -y jq
                        ;;
                    yq)
                        # yq는 snap 또는 직접 다운로드
                        if command -v snap &> /dev/null; then
                            snap install yq
                        else
                            wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
                            chmod +x /usr/local/bin/yq
                        fi
                        ;;
                    terraform)
                        log_warn "Terraform은 수동 설치가 필요합니다."
                        log_info "https://developer.hashicorp.com/terraform/downloads 참조"
                        ;;
                esac
            done
        elif [[ -f /etc/redhat-release ]]; then
            for pkg in "${missing[@]}"; do
                case $pkg in
                    jq)
                        yum install -y jq || dnf install -y jq
                        ;;
                    yq)
                        wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
                        chmod +x /usr/local/bin/yq
                        ;;
                    terraform)
                        log_warn "Terraform은 수동 설치가 필요합니다."
                        log_info "https://developer.hashicorp.com/terraform/downloads 참조"
                        ;;
                esac
            done
        else
            log_error "지원하지 않는 OS입니다. 패키지를 수동으로 설치하세요."
        fi
    fi

    log_success "필수 패키지 확인 완료"
}

# 디렉토리 생성
create_directories() {
    log_info "디렉토리 생성 중..."

    # 데이터 디렉토리
    mkdir -p /var/lib/basphere/{ipam,terraform,clusters,users,key-requests,templates}

    # 로그 디렉토리
    mkdir -p /var/log/basphere

    # 설정 디렉토리
    mkdir -p /etc/basphere

    log_success "디렉토리 생성 완료"
}

# 그룹 생성
create_groups() {
    log_info "그룹 생성 중..."

    # basphere-users 그룹 (일반 사용자)
    if ! getent group basphere-users > /dev/null; then
        groupadd basphere-users
        log_success "basphere-users 그룹 생성 완료"
    else
        log_info "basphere-users 그룹이 이미 존재합니다"
    fi

    # basphere-admin 그룹 (관리자)
    if ! getent group basphere-admin > /dev/null; then
        groupadd basphere-admin
        log_success "basphere-admin 그룹 생성 완료"
    else
        log_info "basphere-admin 그룹이 이미 존재합니다"
    fi
}

# 서비스 계정 생성
create_service_account() {
    log_info "서비스 계정 생성 중..."

    # basphere 서비스 계정 (스크립트 실행용)
    if ! id "basphere" &> /dev/null; then
        useradd -r -s /bin/bash -d /var/lib/basphere -c "Basphere Service Account" basphere
        log_success "basphere 서비스 계정 생성 완료"
    else
        log_info "basphere 서비스 계정이 이미 존재합니다"
    fi
}

# IPAM 초기화
init_ipam() {
    log_info "IPAM 초기화 중..."

    local ipam_dir="/var/lib/basphere/ipam"

    # allocations.tsv - 사용자별 IP 블록 할당
    if [[ ! -f "$ipam_dir/allocations.tsv" ]]; then
        echo -e "# user\tblock_start\tallocated_at" > "$ipam_dir/allocations.tsv"
        log_success "allocations.tsv 생성 완료"
    fi

    # leases.tsv - 개별 IP 할당
    if [[ ! -f "$ipam_dir/leases.tsv" ]]; then
        echo -e "# ip\tuser\tresource_name\tresource_type\tallocated_at" > "$ipam_dir/leases.tsv"
        log_success "leases.tsv 생성 완료"
    fi

    # 락 파일
    touch "$ipam_dir/.lock"

    log_success "IPAM 초기화 완료"
}

# 설정 파일 복사
copy_config_files() {
    log_info "설정 파일 복사 중..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # config.yaml
    if [[ ! -f /etc/basphere/config.yaml ]]; then
        if [[ -f "$script_dir/config/config.yaml.example" ]]; then
            cp "$script_dir/config/config.yaml.example" /etc/basphere/config.yaml
            log_success "config.yaml 복사 완료 (수정 필요)"
        else
            log_warn "config.yaml.example을 찾을 수 없습니다"
        fi
    else
        log_info "config.yaml이 이미 존재합니다"
    fi

    # specs.yaml
    if [[ ! -f /etc/basphere/specs.yaml ]]; then
        if [[ -f "$script_dir/config/specs.yaml.example" ]]; then
            cp "$script_dir/config/specs.yaml.example" /etc/basphere/specs.yaml
            log_success "specs.yaml 복사 완료"
        else
            log_warn "specs.yaml.example을 찾을 수 없습니다"
        fi
    else
        log_info "specs.yaml이 이미 존재합니다"
    fi

    # vsphere.env
    if [[ ! -f /etc/basphere/vsphere.env ]]; then
        if [[ -f "$script_dir/config/vsphere.env.example" ]]; then
            cp "$script_dir/config/vsphere.env.example" /etc/basphere/vsphere.env
            chmod 600 /etc/basphere/vsphere.env
            log_success "vsphere.env 복사 완료 (수정 필요, 권한 600)"
        else
            log_warn "vsphere.env.example을 찾을 수 없습니다"
        fi
    else
        log_info "vsphere.env가 이미 존재합니다"
    fi
}

# 템플릿 복사
copy_templates() {
    log_info "템플릿 복사 중..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local template_dir="/var/lib/basphere/templates"

    # Terraform 템플릿
    if [[ -d "$script_dir/templates/terraform" ]]; then
        cp -r "$script_dir/templates/terraform" "$template_dir/"
        log_success "Terraform 템플릿 복사 완료"
    fi

    # CAPI 템플릿 (Stage 2)
    if [[ -d "$script_dir/templates/capi" ]]; then
        cp -r "$script_dir/templates/capi" "$template_dir/"
        log_success "CAPI 템플릿 복사 완료"
    fi
}

# CLI 스크립트 설치
install_cli_scripts() {
    log_info "CLI 스크립트 설치 중..."

    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local bin_dir="/usr/local/bin"

    # lib 복사
    mkdir -p /usr/local/lib/basphere
    if [[ -d "$script_dir/lib" ]]; then
        cp -r "$script_dir/lib/"* /usr/local/lib/basphere/
        log_success "라이브러리 복사 완료"
    fi

    # 내부 스크립트 복사
    mkdir -p /usr/local/lib/basphere/internal
    if [[ -d "$script_dir/scripts/internal" ]]; then
        cp "$script_dir/scripts/internal/"* /usr/local/lib/basphere/internal/
        chmod +x /usr/local/lib/basphere/internal/*
        log_success "내부 스크립트 복사 완료"
    fi

    # 관리자 CLI
    if [[ -f "$script_dir/scripts/basphere-admin" ]]; then
        cp "$script_dir/scripts/basphere-admin" "$bin_dir/"
        chmod +x "$bin_dir/basphere-admin"
        log_success "basphere-admin 설치 완료"
    fi

    # 사용자 CLI
    local user_scripts=("create-vm" "delete-vm" "list-vms" "list-resources" "show-quota")
    for script in "${user_scripts[@]}"; do
        if [[ -f "$script_dir/scripts/user/$script" ]]; then
            cp "$script_dir/scripts/user/$script" "$bin_dir/"
            chmod +x "$bin_dir/$script"
            log_success "$script 설치 완료"
        fi
    done
}

# 권한 설정
set_permissions() {
    log_info "권한 설정 중..."

    # 데이터 디렉토리 기본 소유권
    chown -R basphere:basphere /var/lib/basphere

    # 기본 디렉토리 권한 (읽기 가능)
    chmod 755 /var/lib/basphere
    chmod 755 /var/lib/basphere/users
    chmod 755 /var/lib/basphere/clusters
    chmod 755 /var/lib/basphere/templates
    chmod -R 755 /var/lib/basphere/templates/terraform 2>/dev/null || true

    # terraform 디렉토리 (사용자가 VM 디렉토리 생성 가능해야 함)
    chmod 755 /var/lib/basphere/terraform
    find /var/lib/basphere/terraform -mindepth 1 -maxdepth 1 -type d -exec chmod 777 {} \; 2>/dev/null || true

    # IPAM 디렉토리 (사용자가 락 획득 및 파일 쓰기 가능해야 함)
    chmod 777 /var/lib/basphere/ipam
    chmod 666 /var/lib/basphere/ipam/.lock 2>/dev/null || true
    chmod 644 /var/lib/basphere/ipam/allocations.tsv 2>/dev/null || true
    chmod 666 /var/lib/basphere/ipam/leases.tsv 2>/dev/null || true

    # 사용자 디렉토리 권한
    find /var/lib/basphere/users -mindepth 1 -maxdepth 1 -type d -exec chmod 755 {} \; 2>/dev/null || true
    find /var/lib/basphere/users -name "metadata.json" -exec chmod 644 {} \; 2>/dev/null || true

    # 로그 디렉토리 권한 (사용자가 감사 로그 쓰기 가능해야 함)
    chown -R basphere:basphere /var/log/basphere
    chmod 777 /var/log/basphere
    touch /var/log/basphere/audit.log
    chmod 666 /var/log/basphere/audit.log

    # 설정 디렉토리 권한
    chown -R root:basphere-admin /etc/basphere
    chmod 755 /etc/basphere
    chmod 644 /etc/basphere/config.yaml 2>/dev/null || true
    chmod 644 /etc/basphere/specs.yaml 2>/dev/null || true
    # vsphere.env는 vSphere 인증정보 포함 - 사용자가 Terraform 실행을 위해 읽기 필요
    chmod 644 /etc/basphere/vsphere.env 2>/dev/null || true

    log_success "권한 설정 완료"
}

# sudoers 설정
setup_sudoers() {
    log_info "sudoers 설정 중..."

    local sudoers_file="/etc/sudoers.d/basphere"

    cat > "$sudoers_file" << 'EOF'
# Basphere CLI sudoers 설정

# basphere-users 그룹: 사용자 CLI 실행 가능
%basphere-users ALL=(basphere) NOPASSWD: /usr/local/bin/create-vm
%basphere-users ALL=(basphere) NOPASSWD: /usr/local/bin/delete-vm
%basphere-users ALL=(basphere) NOPASSWD: /usr/local/bin/list-vms
%basphere-users ALL=(basphere) NOPASSWD: /usr/local/bin/list-resources
%basphere-users ALL=(basphere) NOPASSWD: /usr/local/bin/show-quota

# basphere-admin 그룹: 관리자 CLI 실행 가능
%basphere-admin ALL=(root) NOPASSWD: /usr/local/bin/basphere-admin
EOF

    chmod 440 "$sudoers_file"

    # 문법 검사
    if visudo -cf "$sudoers_file" &> /dev/null; then
        log_success "sudoers 설정 완료"
    else
        log_error "sudoers 파일 문법 오류"
        rm -f "$sudoers_file"
        exit 1
    fi
}

# 설치 완료 메시지
print_completion_message() {
    echo ""
    echo "=========================================="
    echo -e "${GREEN}Basphere CLI 설치가 완료되었습니다!${NC}"
    echo "=========================================="
    echo ""
    echo "다음 단계:"
    echo ""
    echo "1. vSphere 설정 수정:"
    echo "   sudo vim /etc/basphere/config.yaml"
    echo ""
    echo "2. vSphere 인증 정보 설정:"
    echo "   sudo vim /etc/basphere/vsphere.env"
    echo ""
    echo "3. 관리자 계정을 basphere-admin 그룹에 추가:"
    echo "   sudo usermod -aG basphere-admin <your-username>"
    echo ""
    echo "4. 사용자 추가 (관리자가 실행):"
    echo "   sudo basphere-admin user add <username> --pubkey <pubkey-file>"
    echo ""
    echo "5. 사용자가 VM 생성:"
    echo "   create-vm"
    echo ""
}

# 메인 함수
main() {
    echo ""
    echo "=========================================="
    echo "      Basphere CLI 설치 스크립트"
    echo "=========================================="
    echo ""

    check_root
    check_dependencies
    create_directories
    create_groups
    create_service_account
    init_ipam
    copy_config_files
    copy_templates
    install_cli_scripts
    set_permissions
    setup_sudoers
    print_completion_message
}

main "$@"
