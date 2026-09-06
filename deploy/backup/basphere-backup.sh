#!/bin/bash
#
# Basphere 데이터 백업 스크립트
# 사용자 메타데이터, Terraform 상태, IPAM, 설정, SSH 키, 계정 정보를 백업합니다.
#
# 사용법: sudo basphere-backup [--full]
#   --full: 재생성 가능한 산출물(.terraform 프로바이더 캐시, OS 이미지)까지 포함한 전체 백업
#           아카이브가 수 GB 단위로 커지므로 디스크 여유를 확인하고 수동으로만 사용할 것
#
# 설치 위치: /usr/local/sbin/basphere-backup
# 자동 실행: basphere-backup.timer (매일 03:00)
#

set -euo pipefail

BACKUP_DIR="/var/backups/basphere"
RETENTION_DAYS=14
MIN_FREE_MB=1024

log_info() { echo "[INFO] $1"; }
log_warn() { echo "[WARN] $1" >&2; }
log_error() { echo "[ERROR] $1" >&2; }
log_success() { echo "[SUCCESS] $1"; }

if [[ $EUID -ne 0 ]]; then
    log_error "이 스크립트는 root 권한으로 실행해야 합니다: sudo $0"
    exit 1
fi

FULL=false
[[ "${1:-}" == "--full" ]] && FULL=true

# 아카이브가 생성 도중에도 world-readable 상태가 되지 않도록 umask를 먼저 조인다.
# (이전에는 tar 이후 chmod 600에 의존해서, tar와 chmod 사이에 죽은 백업이 644로 남았다)
umask 077

timestamp=$(date +%Y%m%d-%H%M%S)
archive="$BACKUP_DIR/basphere-${timestamp}.tar.gz"
workdir=$(mktemp -d)
archive_done=false
cleanup() {
    rm -rf "$workdir"
    # 중간에 죽었으면 불완전한 아카이브를 남기지 않는다 (디스크만 먹고 복원에는 쓸 수 없음)
    $archive_done || rm -f "$archive"
}
trap cleanup EXIT

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"

# 보존 기간이 지난 백업 삭제 — 반드시 tar보다 "먼저" 실행한다.
# tar 뒤에 두면 디스크가 찬 순간 tar가 죽으면서 정리 로직에 도달하지 못하고,
# 다음 날도 같은 이유로 죽는 자기강화 루프에 빠진다 (2026-08 장애의 실제 원인).
deleted=$(find "$BACKUP_DIR" -name "basphere-*.tar.gz" -mtime +"$RETENTION_DAYS" -print -delete | wc -l) || deleted=0
[[ "$deleted" -gt 0 ]] && log_info "보존 기간(${RETENTION_DAYS}일) 경과 백업 ${deleted}개 삭제"

# 0바이트 아카이브(디스크 풀 등으로 실패한 잔해)도 함께 정리
find "$BACKUP_DIR" -name "basphere-*.tar.gz" -size 0 -delete || true

free_mb=$(df -Pm "$BACKUP_DIR" | awk 'NR==2 {print $4}')
if [[ "$free_mb" -lt "$MIN_FREE_MB" ]]; then
    log_error "디스크 여유 공간 부족: ${free_mb}MB (최소 ${MIN_FREE_MB}MB 필요) — 백업을 중단합니다"
    exit 1
fi

# 계정/그룹 정보 스냅샷 (사용자 계정 복구용)
getent passwd > "$workdir/accounts-passwd.txt"
getent group > "$workdir/accounts-groups.txt"

# 백업 대상 수집 (경로는 / 기준 상대 경로)
targets=(var/lib/basphere etc/basphere)
[[ -f /etc/sudoers.d/basphere ]] && targets+=(etc/sudoers.d/basphere)
for ssh_dir in /home/*/.ssh; do
    [[ -d "$ssh_dir" ]] && targets+=("${ssh_dir#/}")
done

# 재생성 가능한 산출물은 기본 제외:
#   .terraform            → terraform init으로 복구 가능 (약 936MB)
#   var/lib/basphere/images → OS 이미지 빌드 산출물, 재다운로드 가능 (약 4.8GB)
# images를 포함하면 아카이브 1개가 4.8GB가 되어 14일 보존(=67GB)이 58GB 디스크에
# 원천적으로 들어갈 수 없다. 제외 시 아카이브는 약 1.4MB.
exclude_args=()
if ! $FULL; then
    exclude_args+=(--exclude='.terraform' --exclude='var/lib/basphere/images')
fi

log_info "백업 생성 중: $archive"
tar_rc=0
tar czf "$archive" "${exclude_args[@]}" \
    -C / "${targets[@]}" \
    -C "$workdir" accounts-passwd.txt accounts-groups.txt || tar_rc=$?

# GNU tar는 "file changed as we read it" 같은 비치명적 경고에도 exit 1을 반환한다.
# 이 경우 아카이브 자체는 유효하므로 실패로 처리하지 않는다. exit 2 이상만 치명적.
if [[ "$tar_rc" -eq 1 ]]; then
    log_warn "tar가 경고와 함께 종료됨 (exit 1) — 아카이브는 생성되었으나 일부 파일이 읽는 중 변경되었을 수 있습니다"
elif [[ "$tar_rc" -ne 0 ]]; then
    log_error "백업 실패: tar가 exit ${tar_rc}로 종료 (불완전한 아카이브는 삭제됨)"
    exit 1
fi

archive_done=true
chmod 600 "$archive"

log_success "백업 완료: $archive ($(du -h "$archive" | cut -f1))"
