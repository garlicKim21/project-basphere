# Basphere - Self-Service Infrastructure Platform

## 프로젝트 개요

Basphere는 VMware vSphere 기반의 셀프서비스 인프라 플랫폼입니다.
개발자가 Bastion 서버에 SSH 접속하여 직접 VM을 생성/관리할 수 있습니다.

**최종 목표**: Backstage 기반 IDP(Internal Developer Platform) 구축

## 아키텍처

```
┌─────────────┐     SSH      ┌─────────────┐                 ┌─────────────┐
│  Developer  │─────────────▶│   Bastion   │                 │   vSphere   │
│  (MacBook)  │              │             │                 │  (vCenter)  │
└─────────────┘              └──────┬──────┘                 └─────────────┘
                             CLI    │                               ▲
                           (HTTP)   ▼                               │
                             ┌─────────────┐    Terraform    ───────┘
                             │ API Server  │
                             │   (root)    │
                             └─────────────┘
```

**보안**: CLI는 API 서버를 통해 VM 작업 수행, vSphere 인증 정보는 root만 접근 가능

## 디렉토리 구조

```
project-basphere/
├── basphere-cli/           # Bash 기반 CLI 도구
│   ├── scripts/            # CLI 스크립트 (basphere-admin, user/)
│   ├── lib/common.sh       # 공통 함수
│   └── templates/terraform/ # Terraform 템플릿
│
├── basphere-api/           # Go REST API 서버
│   ├── internal/           # handler, model, store, provisioner
│   └── web/templates/      # HTML 템플릿
│
├── docs/                   # 📚 문서
│   ├── design/             # IDP 설계 (vision, architecture, roadmap)
│   ├── operations/         # 운영 (installation, troubleshooting, security)
│   └── development/        # 개발 (contributing)
│
└── deploy/                 # nginx, systemd 설정
```

## 기술 스택

| 구분 | 현재 (Stage 1) | 목표 (Stage 3 - IDP) |
|------|---------------|---------------------|
| CLI | Bash, jq, yq | - |
| API | Go 1.21+, chi router | Go + PostgreSQL |
| IaC | Terraform + vSphere | Crossplane |
| K8s | - | Cluster API |
| 포털 | 웹 폼 | Backstage |
| GitOps | - | ArgoCD / Flux |

## 프로젝트 상태

### Stage 1 (MVP) - ✅ 완료

- 사용자 관리 (등록/승인/삭제)
- VM 생성/조회/삭제
- 다중 OS (Ubuntu 24.04, Rocky 10.1)
- IP 자동 할당 (IPAM)
- API 기반 아키텍처
- 보안 (SSH 키 인증, fail2ban)

### Stage 2 - 🚧 예정

- Kubernetes 클러스터 프로비저닝 (Cluster API)
- 테넌트 네트워크 격리

### Stage 3 (IDP) - 📋 계획

- Backstage 포털
- Crossplane 인프라 제어
- GitOps (ArgoCD/Flux)
- Harbor, CI/CD 통합

## 개발 규칙

### Bash
- `set -euo pipefail` 필수
- 함수명: snake_case
- 로그: `log_info`, `log_success`, `log_warn`, `log_error`

### Go
- `gofmt` 적용
- 인터페이스로 추상화 (Store, Provisioner)

### 커밋
- Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`

## 주요 설정 파일 (Bastion)

| 파일 | 설명 |
|------|------|
| `/etc/basphere/config.yaml` | 메인 설정 (vSphere, 네트워크) |
| `/etc/basphere/vsphere.env` | vSphere 인증 **(600 권한)** |
| `/etc/basphere/api.yaml` | API 서버 설정 |
| `/etc/basphere/specs.yaml` | VM 스펙 정의 |
| `/var/lib/basphere/` | 데이터 디렉토리 |

## 자주 사용하는 명령어

### 로컬
```bash
git add -A && git commit -m "message" && git push
cd basphere-api && make build-linux
```

### Bastion
```bash
# 코드 업데이트 및 CLI 재설치
cd /opt/basphere && sudo git pull
cd /opt/basphere/basphere-cli && sudo ./install.sh

# 사용자 관리
sudo basphere-admin user list
sudo basphere-admin user approve <username>

# VM 테스트
create-vm -n test -s small
list-vms
delete-vm test
```

## 주의사항

- vSphere customization과 cloud-init 함께 사용 시 네트워크 설정 충돌 주의
- snap yq는 /etc 접근 불가 → 바이너리 버전 사용
- **Ubuntu 24.04 cloud-init**: 네트워크 설정은 `guestinfo.metadata` 안에 `network` 키로 포함

## 📚 상세 문서

| 문서 | 설명 |
|------|------|
| [docs/design/vision.md](docs/design/vision.md) | 프로젝트 비전 및 목표 |
| [docs/design/architecture.md](docs/design/architecture.md) | 전체 아키텍처 |
| [docs/design/roadmap.md](docs/design/roadmap.md) | Stage별 상세 계획 |
| [docs/operations/installation.md](docs/operations/installation.md) | 새 환경 설치 가이드 |
| [docs/operations/troubleshooting.md](docs/operations/troubleshooting.md) | 트러블슈팅 |
| [docs/operations/security.md](docs/operations/security.md) | 보안 설정 |
| [docs/development/contributing.md](docs/development/contributing.md) | 개발 규칙 |
| [basphere-cli/README.md](basphere-cli/README.md) | CLI 가이드 |
| [basphere-api/README.md](basphere-api/README.md) | API 가이드 |
