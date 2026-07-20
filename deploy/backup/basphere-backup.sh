#!/bin/bash
#
# Basphere 데이터 백업 스크립트
# 사용자 메타데이터, Terraform 상태, IPAM, 설정, SSH 키, 계정 정보를 백업합니다.
#
# 사용법: sudo basphere-backup [--full]
#   --full: Terraform 프로바이더 캐시(.terraform)까지 포함한 전체 백업
#
# 설치 위치: /usr/local/sbin/basphere-backup
# 자동 실행: basphere-backup.timer (매일 03:00)
#

set -euo pipefail

BACKUP_DIR="/var/backups/basphere"
RETENTION_DAYS=14

log_info() { echo "[INFO] $1"; }
log_error() { echo "[ERROR] $1" >&2; }
log_success() { echo "[SUCCESS] $1"; }

if [[ $EUID -ne 0 ]]; then
    log_error "이 스크립트는 root 권한으로 실행해야 합니다: sudo $0"
    exit 1
fi

FULL=false
[[ "${1:-}" == "--full" ]] && FULL=true

timestamp=$(date +%Y%m%d-%H%M%S)
archive="$BACKUP_DIR/basphere-${timestamp}.tar.gz"
workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"

# 계정/그룹 정보 스냅샷 (사용자 계정 복구용)
getent passwd > "$workdir/accounts-passwd.txt"
getent group > "$workdir/accounts-groups.txt"

# 백업 대상 수집 (경로는 / 기준 상대 경로)
targets=(var/lib/basphere etc/basphere)
[[ -f /etc/sudoers.d/basphere ]] && targets+=(etc/sudoers.d/basphere)
for ssh_dir in /home/*/.ssh; do
    [[ -d "$ssh_dir" ]] && targets+=("${ssh_dir#/}")
done

# .terraform(프로바이더 캐시)은 terraform init으로 재생성 가능하므로 기본 제외
exclude_args=()
$FULL || exclude_args+=(--exclude='.terraform')

log_info "백업 생성 중: $archive"
tar czf "$archive" "${exclude_args[@]}" \
    -C / "${targets[@]}" \
    -C "$workdir" accounts-passwd.txt accounts-groups.txt
chmod 600 "$archive"

# 보존 기간이 지난 백업 삭제
deleted=$(find "$BACKUP_DIR" -name "basphere-*.tar.gz" -mtime +"$RETENTION_DAYS" -print -delete | wc -l)
[[ "$deleted" -gt 0 ]] && log_info "보존 기간(${RETENTION_DAYS}일) 경과 백업 ${deleted}개 삭제"

log_success "백업 완료: $archive ($(du -h "$archive" | cut -f1))"
