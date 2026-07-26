# Basphere 로드맵

## 개요

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
│  Stage 2 🚧 진행 중 (코드 구현 완료 / Management Cluster 미구축)              │
│  ├── Kubernetes 클러스터 프로비저닝 (Cluster API) — CLI/API 구현 완료          │
│  └── 테넌트 네트워크 격리 — 미착수                                            │
│                                                                             │
│  Stage 3 (IDP)                                                               │
│  ├── Backstage 기반 포털                                                     │
│  ├── Crossplane 인프라 제어                                                  │
│  ├── GitOps (ArgoCD/Flux)                                                   │
│  └── Harbor, CI/CD 통합                                                     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Stage 1: MVP (CLI 기반 VM 셀프서비스)

**상태**: ✅ 완료 (2026-01-19)

### 완료된 기능

- [x] 사용자 관리 (생성/삭제)
- [x] 웹 기반 사용자 등록 요청
- [x] 관리자 승인/거부 (CLI)
- [x] VM 생성/조회/삭제
- [x] IP 자동 할당 (IPAM)
- [x] MTU 설정 지원
- [x] 의존성 자동 설치
- [x] 다중 OS 지원 (Ubuntu 24.04, Rocky Linux 10.1)
- [x] OS별 네트워크 인터페이스 자동 설정
- [x] 디스크 자동 확장 (growpart)
- [x] 5단계 VM 스펙 (tiny, small, medium, large, huge)
- [x] API 기반 VM 관리 - CLI → API → Terraform 아키텍처
- [x] vSphere 인증 정보 보호 - vsphere.env 600 권한
- [x] nginx 리버스 프록시 - 내부 API 외부 차단
- [x] Google reCAPTCHA v2 - 등록 폼 봇 방지
- [x] SSH 키 변경 요청 - 웹 폼 + 관리자 승인 워크플로우
- [x] 이메일 도메인 검증 - 허용된 도메인만 등록 가능
- [x] SSH 보안 강화 - 비밀번호 인증 비활성화, fail2ban
- [x] 외부 접근 지원 - bastion 주소/포트 설정 가능

### VM 스펙

| 스펙 | vCPU | RAM | Disk | 용도 |
|------|------|-----|------|------|
| tiny | 2 | 4GB | 50GB | 테스트용 |
| small | 2 | 8GB | 50GB | 개발용 |
| medium | 4 | 16GB | 100GB | 일반 워크로드 |
| large | 8 | 32GB | 200GB | 고성능 워크로드 |
| huge | 16 | 64GB | 200GB | 대규모 워크로드 |

### 지원 OS

| OS | 템플릿 | 인터페이스 |
|----|--------|-----------|
| Ubuntu 24.04 LTS | ubuntu-noble-24.04-cloudimg | ens192 |
| Rocky Linux 10.1 | rocky-10-template | ens33 |

### 아키텍처

```
┌─────────────┐     SSH      ┌─────────────┐                 ┌─────────────┐
│  Developer  │─────────────▶│   Bastion   │                 │   vSphere   │
│             │              │             │                 │  (vCenter)  │
└─────────────┘              └──────┬──────┘                 └─────────────┘
       │                           │                                ▲
       │      HTTP (등록 폼)        │                                │
       └───────────────────────────┤                                │
                                   ▼                                │
                            ┌─────────────┐    Terraform    ────────┘
                            │  API Server │
                            │  (Go/chi)   │
                            └─────────────┘
```

---

## Stage 2: Kubernetes 클러스터 프로비저닝

**상태**: 🚧 진행 중 — 코드 구현 완료 / 실환경 미가동

CLI·API·CAPI 템플릿은 모두 구현되어 Bastion에 설치까지 되어 있으나,
Management Cluster(kind)가 구축되지 않아 실제 클러스터 생성은 아직 불가능합니다.

### 목표

- [x] Kubernetes 클러스터 셀프서비스 생성 — CLI/API 구현 완료 (실행 환경 미구축)
- [ ] Management Cluster 구축 및 end-to-end 검증
- [ ] 테넌트 네트워크 격리

### 구현 현황

| 영역 | 구현물 | 상태 |
|------|--------|------|
| 사용자 CLI | `create-cluster`, `list-clusters`, `delete-cluster`, `watch-cluster`, `get-kubeconfig` | ✅ 구현·설치 완료 |
| 공통 라이브러리 | `lib/cluster-common.sh` (타입/할당량/메타데이터/kubeconfig 추출) | ✅ |
| 관리자 CLI | `setup-management-cluster` (Docker·kind·kubectl·clusterctl 설치, CAPI/CAPV init) | ✅ 구현 완료 / ❌ 미실행 |
| API 서버 | `POST/GET /api/v1/clusters`, `/clusters/{name}`, `/quota`, `/status`, `/kubeconfig` (7개 엔드포인트) | ✅ |
| 데이터 모델 | `model.Cluster`, 상태 5종 (pending/provisioning/ready/deleting/failed) | ✅ |
| CAPI 템플릿 | `templates/capi/cluster.yaml.tmpl` (Cluster, VSphereCluster, KubeadmControlPlane, MachineDeployment, InClusterIPPool) | ✅ |
| 권한/저장소 | sudoers 등록, `/var/lib/basphere/clusters/<user>/` 사용자별 디렉토리 | ✅ |
| Management Cluster | kind 기반 CAPI/CAPV 실행 환경 | ❌ 미구축 |

### 클러스터 타입

| 타입 | Control Plane | Worker | 용도 |
|------|--------------|--------|------|
| dev | 1대 (small) | 2대 (small) | 개발/테스트 |
| standard | 3대 (medium) | 3대 (large) | HA 운영 환경 |

Worker 스펙은 `create-cluster -w {small\|medium\|large}`로 재정의 가능합니다.

### 기술 스택 (구현 기준)

| 구성 요소 | 계획 | 실제 구현 |
|----------|------|----------|
| 프로비저닝 | Cluster API (CAPV) | Cluster API v1.6.0 + CAPV ✅ |
| Management Cluster | kind | kind (`setup-management-cluster`) ✅ |
| 노드 이미지 | Talos Linux / Ubuntu | Ubuntu + kubeadm (`ubuntu-2204-kube-v1.28.0`) |
| Kubernetes 버전 | - | v1.28.0 (템플릿에 고정) |
| CNI | Cilium | **Calico v3.27.0** (kubeadm postCommand 설치) |
| IPAM | - | CAPI InClusterIPPool + Basphere IPAM 연동 |
| CSI | VMware CSI, Synology NFS | ❌ 미구현 |

### 남은 작업

1. **Management Cluster 구축** — Bastion에 Docker/kind/kubectl/clusterctl 미설치 상태.
   `sudo setup-management-cluster` 실행 및 vSphere 인증 연동 필요
2. **specs.yaml 키 정합성 수정** — 코드는 `.cluster_types.*` / `.cluster_node_specs.*`를 조회하는데
   `specs.yaml`은 `cluster_specs:`로 정의되어 있어 전부 기본값으로 폴백.
   결과적으로 `standard` 선택이 `dev`와 동일하게 동작하고 대화형 타입 선택은 실패
3. **K8s 노드 이미지 준비** — `config.yaml`에 `templates.kubernetes` 항목이 없어 기본값 사용.
   kubeadm 사전 설치된 OVA를 컨텐츠 라이브러리에 등록 필요
4. **end-to-end 검증** — 현재 생성된 클러스터 0개
5. CSI(스토리지), CNI 선택지(Cilium), 테넌트 네트워크 격리

### 아키텍처

```
┌─────────────────────────────────────────────┐
│            Management Cluster               │
│  ┌─────────────┐  ┌─────────────┐          │
│  │ Cluster API │  │    CAPV     │          │
│  │  Operator   │  │  (vSphere)  │          │
│  └──────┬──────┘  └──────┬──────┘          │
│         │                │                  │
│         └────────┬───────┘                  │
└──────────────────┼──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│         Workload Clusters (User)            │
│  ┌─────────────┐  ┌─────────────┐          │
│  │   Dev K8s   │  │  Prod K8s   │          │
│  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────┘
```

---

## Stage 3: IDP (Internal Developer Platform)

**상태**: 📋 계획

### 목표

- [ ] Backstage 기반 포털
- [ ] Crossplane 인프라 제어
- [ ] GitOps (ArgoCD/Flux)
- [ ] Harbor 컨테이너 레지스트리
- [ ] CI/CD 통합

### 계획된 기능

#### 사용자 인터페이스

- Backstage 포털
- 서비스 카탈로그
- 셀프서비스 워크플로우

#### 인프라 자동화

- Crossplane으로 vSphere 리소스 관리
- GitOps 기반 선언적 인프라

#### 개발자 경험

- 템플릿 기반 프로젝트 생성
- 통합 CI/CD 파이프라인
- 통합 모니터링/로깅

### 기술 스택

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

---

## 관련 문서

- [비전](vision.md) - 프로젝트 목표
- [아키텍처](architecture.md) - 전체 시스템 아키텍처
- [사용자 시나리오](user-scenarios.md) - 사용자 워크플로우
- [인프라](infrastructure.md) - 하드웨어 사양
