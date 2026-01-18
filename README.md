# Project Basphere

개발자 대상 셀프서비스 인프라 플랫폼

## 개요

Basphere는 VMware vSphere 기반의 셀프서비스 인프라 플랫폼입니다. 개발자가 VM과 Kubernetes 클러스터를 직접 프로비저닝하고 관리할 수 있는 환경을 제공합니다.

## 로드맵

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Project Basphere                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Stage 1 (MVP) ✅ 완료                                                       │
│  ├── CLI 기반 VM 셀프서비스                                                   │
│  ├── 웹 기반 사용자 등록                                                      │
│  └── 관리자 CLI                                                              │
│                                                                             │
│  Stage 2 (예정)                                                              │
│  ├── Kubernetes 클러스터 프로비저닝 (Cluster API)                             │
│  └── 테넌트 네트워크 격리                                                     │
│                                                                             │
│  Stage 3 (IDP)                                                               │
│  ├── Backstage 기반 포털                                                     │
│  ├── Crossplane 인프라 제어                                                  │
│  ├── GitOps (ArgoCD/Flux)                                                   │
│  └── Harbor, CI/CD 통합                                                     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 현재 상태: Stage 1 (MVP)

CLI와 간단한 API를 통한 VM 셀프서비스를 제공합니다.

### 아키텍처

```
┌─────────────┐     SSH      ┌─────────────┐    Terraform    ┌─────────────┐
│  Developer  │─────────────▶│   Bastion   │────────────────▶│   vSphere   │
│             │              │  (CLI/API)  │                 │  (vCenter)  │
└─────────────┘              └─────────────┘                 └─────────────┘
       │                           │
       │      HTTP (등록 폼)        │
       └───────────────────────────┘
```

### 주요 기능

| 기능 | 설명 |
|------|------|
| VM 프로비저닝 | CLI로 VM 생성/조회/삭제 |
| 다중 OS 지원 | Ubuntu 24.04, Rocky Linux 10.1 |
| IP 자동 할당 | 사용자별 IP 블록, 경량 IPAM |
| 사용자 등록 | 웹 폼으로 등록 요청, 관리자 승인 |
| 리소스 할당량 | 사용자별 VM/IP 제한 |

### 지원 VM 스펙

| 스펙 | vCPU | RAM | Disk |
|------|------|-----|------|
| tiny | 2 | 4GB | 50GB |
| small | 2 | 8GB | 50GB |
| medium | 4 | 16GB | 100GB |
| large | 8 | 32GB | 200GB |
| huge | 16 | 64GB | 200GB |

## 디렉토리 구조

```
project-basphere/
├── basphere-cli/           # Bash 기반 CLI 도구
│   ├── scripts/            # CLI 스크립트
│   ├── lib/                # 공통 함수 라이브러리
│   ├── templates/          # Terraform 템플릿
│   ├── config/             # 설정 파일 예시
│   └── docs/               # 사용자 가이드
│
├── basphere-api/           # Go 기반 REST API 서버
│   ├── cmd/                # 서버 진입점
│   ├── internal/           # 내부 패키지
│   └── web/                # HTML 템플릿
│
├── project-base/           # IDP 설계 문서
│   ├── architecture.yaml   # 아키텍처 설계
│   ├── user-scenario.yaml  # 사용자 시나리오
│   └── project-concept.yaml# 프로젝트 컨셉
│
└── CLAUDE.md               # 개발 컨텍스트 (AI 협업용)
```

## 기술 스택

### 현재 (Stage 1)

| 구분 | 기술 |
|------|------|
| CLI | Bash, jq, yq |
| API | Go 1.21+, chi router |
| IaC | Terraform + vSphere Provider |
| VM 초기화 | cloud-init |
| 스토리지 | 파일 기반 (JSON) |

### 목표 (Stage 3 - IDP)

| 구분 | 기술 |
|------|------|
| 포털 | Backstage |
| 인프라 제어 | Crossplane |
| K8s 프로비저닝 | Cluster API |
| GitOps | ArgoCD / Flux |
| 컨테이너 레지스트리 | Harbor |
| 테넌트 라우터 | OPNsense |
| VPN | WireGuard |
| 데이터베이스 | PostgreSQL |

## 빠른 시작

### 사용자

1. 웹 폼에서 등록 요청 제출
2. 관리자 승인 후 Bastion SSH 접속
3. VM 생성 및 관리

```bash
# Bastion 접속
ssh <username>@<bastion-server>

# VM 생성
create-vm -n my-server -s small

# VM 목록
list-vms

# VM 삭제
delete-vm my-server
```

### 관리자

```bash
# 대기 중인 등록 요청 확인
sudo basphere-admin user pending

# 사용자 승인
sudo basphere-admin user approve <username>

# 사용자 목록
sudo basphere-admin user list
```

## 설치

상세한 설치 가이드는 각 컴포넌트의 README를 참조하세요:

- [CLI 설치 가이드](basphere-cli/README.md)
- [API 서버 가이드](basphere-api/README.md)
- [사용자 가이드](basphere-cli/docs/user-guide.md)

## IDP 비전

Basphere는 현재 CLI 기반 MVP에서 시작하여, 궁극적으로 완전한 Internal Developer Platform(IDP)으로 발전할 계획입니다.

### 목표 아키텍처

```
┌──────────────────────────────────────────────────────────────────────────┐
│                              Backstage Portal                             │
│                     (사용자 포털, 서비스 카탈로그)                          │
└────────────────────────────────┬─────────────────────────────────────────┘
                                 │
┌────────────────────────────────▼─────────────────────────────────────────┐
│                           Control Plane                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │  Crossplane │  │ Cluster API │  │   ArgoCD    │  │   Harbor    │     │
│  │  (인프라)   │  │ (K8s 클러스터)│  │   (GitOps)  │  │  (Registry) │     │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘     │
└────────────────────────────────┬─────────────────────────────────────────┘
                                 │
┌────────────────────────────────▼─────────────────────────────────────────┐
│                         Tenant Infrastructure                             │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │                    Tenant Network (OPNsense)                        │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │ │
│  │  │  K8s Cluster│  │  K8s Cluster│  │    VMs      │                 │ │
│  │  │  (Dev)      │  │  (Prod)     │  │             │                 │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                 │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────┘
```

### 핵심 목표

1. **셀프서비스**: 개발자가 인프라 팀 개입 없이 리소스 프로비저닝
2. **멀티테넌시**: 팀/프로젝트별 격리된 네트워크 환경
3. **표준화**: Golden Path를 통한 일관된 개발 환경
4. **자동화**: GitOps 기반 인프라 및 애플리케이션 배포

## 기여

이 프로젝트는 홈랩 환경에서 IDP 구축을 학습하고 실험하기 위한 목적으로 개발되었습니다.

## 라이선스

MIT License
