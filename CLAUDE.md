# Basphere - Self-Service Infrastructure Platform

## 프로젝트 개요

Basphere는 VMware vSphere 기반의 셀프서비스 인프라 플랫폼입니다.
개발자가 Bastion 서버에 SSH 접속하여 직접 VM을 생성/관리할 수 있습니다.

## 아키텍처

```
┌─────────────┐     SSH      ┌─────────────┐    Terraform    ┌─────────────┐
│  Developer  │─────────────▶│   Bastion   │────────────────▶│   vSphere   │
│  (MacBook)  │              │  (CLI/API)  │                 │  (vCenter)  │
└─────────────┘              └─────────────┘                 └─────────────┘
```

## 디렉토리 구조

```
project-basphere/
├── basphere-cli/           # Bash 기반 CLI 도구
│   ├── scripts/
│   │   ├── basphere-admin  # 관리자 CLI (user add/delete/approve 등)
│   │   ├── user/           # 사용자 CLI (create-vm, delete-vm 등)
│   │   └── internal/       # 내부 스크립트 (IPAM 등)
│   ├── lib/
│   │   └── common.sh       # 공통 함수 라이브러리
│   ├── templates/
│   │   └── terraform/      # Terraform 템플릿 (.tf.tmpl)
│   ├── config/             # 설정 파일 예시
│   └── install.sh          # 설치 스크립트
│
└── basphere-api/           # Go 기반 REST API 서버
    ├── cmd/basphere-api/   # 서버 진입점
    ├── internal/
    │   ├── handler/        # HTTP 핸들러
    │   ├── model/          # 데이터 모델
    │   ├── store/          # 저장소 (파일 기반, 향후 DB)
    │   └── provisioner/    # bash 스크립트 호출
    └── web/templates/      # HTML 템플릿 (등록 폼)
```

## 기술 스택

- **CLI**: Bash, jq, yq
- **API**: Go 1.21+, chi router
- **IaC**: Terraform + vSphere Provider
- **VM 초기화**: cloud-init
- **향후 IDP**: Go 기반, PostgreSQL

## 테스트 환경

| 항목 | 값 |
|------|-----|
| Bastion IP | 172.20.0.10 |
| 관리자 계정 | basphere |
| vCenter | vcenter.home.local |
| VM 네트워크 | 10.254.0.0/21 |
| MTU | 1450 (오버레이 네트워크) |

## 주요 설정 파일 (Bastion)

- `/etc/basphere/config.yaml` - 메인 설정
- `/etc/basphere/vsphere.env` - vSphere 인증 정보
- `/etc/basphere/specs.yaml` - VM 스펙 정의
- `/var/lib/basphere/` - 데이터 디렉토리

## 개발 규칙

### Bash 스크립트
- `set -euo pipefail` 필수
- 함수명: snake_case (예: `create_vm`, `get_config`)
- 로그: `log_info`, `log_success`, `log_warn`, `log_error` 사용
- ShellCheck 경고 없어야 함

### Go 코드
- `gofmt` 적용
- 에러는 반드시 처리
- 인터페이스로 추상화 (Store, Provisioner)

### 커밋 메시지
- Conventional Commits 형식
- 예: `feat: Add MTU configuration support`
- 예: `fix: Use write_files for cloud-init MTU config`

## 자주 사용하는 명령어

### 로컬 (개발)
```bash
# 커밋 및 푸시
git add -A && git commit -m "message" && git push

# Go API 빌드
cd basphere-api && make build-linux
```

### Bastion (테스트)
```bash
# 코드 업데이트 및 재설치
cd /opt/basphere-cli && sudo git pull && sudo ./install.sh

# API 서버 실행 (개발 모드)
cd /opt/basphere-api && sudo ./build/basphere-api-linux-amd64 --dev

# 사용자 관리
sudo basphere-admin user list
sudo basphere-admin user pending
sudo basphere-admin user approve <username>

# VM 테스트 (사용자로)
create-vm -n test -s small
list-vms
delete-vm test
```

## 현재 완료된 기능 (Stage 1.5)

- [x] 사용자 관리 (생성/삭제)
- [x] 웹 기반 사용자 등록 요청
- [x] 관리자 승인/거부 (CLI)
- [x] VM 생성/조회/삭제
- [x] IP 자동 할당 (IPAM)
- [x] MTU 설정 지원
- [x] 의존성 자동 설치

## 다음 계획

- [ ] IDP 구축 (6개월 내)
- [ ] Kubernetes 클러스터 프로비저닝 (Stage 2)
- [ ] 웹 대시보드

## 주의사항

- vSphere customization과 cloud-init을 함께 사용 시 네트워크 설정 충돌 주의
- snap으로 설치된 yq는 /etc 접근 불가 (바이너리 버전 사용)
- Terraform 상태 파일은 로컬 저장 (각 사용자별 디렉토리)
