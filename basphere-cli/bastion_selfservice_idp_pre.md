# Bastion 기반 셀프서비스 VM/Kubernetes 프로비저닝 계획서 (IDP 이전 단계)

---

## 0. 목적

IDP(포털/백스테이지/API) 구축 전 단계에서, 사용자가 스스로(via CLI) VMware vSphere 상에 VM을 만들고, 고정 IP를 부여받고, Cluster API로 Kubernetes 클러스터를 생성할 수 있도록 한다.
관리자는 사용자 온보딩(계정 승인/생성)만 수동으로 수행하고, 나머지는 Bastion에서 제공하는 래핑된 CLI 명령으로 자동화한다.

---

## 1. 핵심 원칙

1. **Bastion 1대 고정**: 셀프서비스의 유일한 제어 지점 (사용자 접근)
2. **Management 클러스터**: Cluster API 실행 환경 (별도 K8s 클러스터)
3. **사용자 ID = Bastion Linux 로컬 계정명**: 인증/권한/감사를 OS 레벨로 단순화
4. **SSH 키 기반 인증**: PasswordAuthentication 비활성화, 키로만 접속
5. **개인키 BYOK 원칙**: 사용자가 로컬에서 키 생성, 공개키만 제출
6. **IP 직접 입력 금지**: 시스템이 자동 할당하여 충돌 제거
7. **VLAN/PortGroup 자동 생성 미실시(현재 단계)**: 단일 Dev 네트워크 + 논리적 IP 블록
8. **선언적 인프라 관리**: Terraform(VM), Cluster API(K8s)로 상태 관리

---

## 2. 아키텍처 개요

```
┌─────────────────────────────────────────────────────────────┐
│              On-Premise Management 클러스터                   │
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │  Cluster API    │  │   Crossplane    │                   │
│  │  (CAPV+Ubuntu)  │  │   (vSphere)     │                   │
│  └────────┬────────┘  └─────────────────┘                   │
└───────────┼─────────────────────────────────────────────────┘
            │
            ▼ K8s 클러스터 생성
┌─────────────────────────────────────────────────────────────┐
│                     Bastion 서버                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  CLI 래퍼                                             │   │
│  │  - create-vm      → Terraform (vSphere Provider)     │   │
│  │  - create-cluster → kubectl apply (CAPI manifest)    │   │
│  │  - delete-vm / delete-cluster / list-resources       │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  상태 저장: /var/lib/basphere/                        │   │
│  │  - ipam/       (IP 할당 상태)                         │   │
│  │  - terraform/  (사용자별 tfstate)                     │   │
│  │  - clusters/   (사용자별 클러스터 메타데이터)           │   │
│  │  - users/      (사용자 메타데이터)                     │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│                  사용자 워크로드 (Dev Network)                │
│  ┌─────────────┐  ┌─────────────────────────────────────┐   │
│  │  개별 VM     │  │  K8s 클러스터 (CAPI + Ubuntu OVA)   │   │
│  │ (Terraform) │  │  Control Plane + Workers            │   │
│  └─────────────┘  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. 환경 가정 (Assumptions)

### 3.1 인프라 환경

- VMware vSphere / vCenter 사용
- On-Premise Management Kubernetes 클러스터 운영 중
  - Cluster API + CAPV (Cluster API Provider vSphere) 설치됨
  - Crossplane + vSphere Provider 설치됨 (선택)
- Bastion VM 구성
  - Terraform CLI 설치 (vSphere Provider)
  - kubectl 설치 (Management 클러스터 접근용 kubeconfig)
  - 사용자 CLI 래퍼 스크립트

### 3.2 VM 템플릿

- Ubuntu 기반 VM 템플릿 (개별 VM용)
  - cloud-init 포함
  - open-vm-tools 포함
  - VMware datasource 설정 (datasource_list: [ VMware ])
  - 템플릿화 전 cloud-init clean --logs 수행
- Kubernetes SIG Ubuntu OVA (Cluster API용)
  - CAPV 호환 이미지
  - https://github.com/kubernetes-sigs/image-builder 참조

### 3.3 네트워크 환경

- Dev 네트워크 단일 L3 구성
  - 네트워크: 10.254.0.0/21 (10.254.0.0 ~ 10.254.7.255)
  - Gateway: 10.254.0.1/21
  - DNS: 8.8.8.8, 1.1.1.1 (또는 사내 DNS)
- 사용자별 주소공간은 논리적 블록으로만 관리
  - 사용자 1명당 32 IP 범위 (/27 크기)
  - 실제 라우팅/서브넷 분리는 하지 않음

---

## 4. 사용자 온보딩 (계정/키) 정책

### 4.1 사용자 ID 규칙

- 회사 사용자에게 Bastion Linux 로컬 계정 부여
- 계정명 규칙: 성+이름 이니셜 (예: 김현태 → kimht)
- 허용 문자: [a-z0-9._-] (소문자 권장)
- 해당 계정명이 모든 감사/리소스 매핑의 기준 ID

### 4.2 SSH 키 정책 (BYOK)

- 사용자는 로컬 PC에서 키 생성
  ```bash
  ssh-keygen -t ed25519
  ```
- 개인키는 사용자만 보관
- 공개키(id_ed25519.pub)만 제출

### 4.3 공개키 제출 방식 (API)

- Bastion에 키 제출용 API 제공 (POST /key-submit)
- 사용자 요청 예시
  ```bash
  curl -X POST https://bastion.company/key-submit \
    -d "user=kimht" \
    -d "pubkey=$(cat ~/.ssh/id_ed25519.pub)"
  ```
- 파일 업로드가 아닌 문자열 전달
- 파일명은 서버가 생성

### 4.4 키 저장 및 감사 규칙

- 저장 경로: /var/lib/basphere/key-requests/
- 파일명 규칙: {user}-{UTC timestamp}.pub
  - 예: kimht-20260109T183012Z.pub
- 권장 추가 로그: 요청자 IP, user, 저장 파일명

### 4.5 관리자 승인 및 계정 생성

- 관리자는 키 요청 검토 후 승인
- 관리자 전용 스크립트로 계정 생성
  ```bash
  sudo basphere-admin user add kimht \
    --pubkey /var/lib/basphere/key-requests/kimht-20260109T183012Z.pub
  ```
- 스크립트 수행 작업
  - useradd 실행
  - .ssh/authorized_keys 구성
  - 소유권/권한 설정
  - 사용자 메타데이터 생성 (/var/lib/basphere/users/{user}/)
  - IP 블록 자동 할당

### 4.6 Bastion SSH 보안 설정 (권장)

```
PasswordAuthentication no
PermitRootLogin no
PubkeyAuthentication yes
```

---

## 5. IP 관리 설계 (경량 IPAM)

### 5.1 설계 목표

- 사용자별 고정 IP 블록(32개 범위) 제공
- 사용자는 IP를 선택하지 않음
- Bastion 1대 기준, 파일 + flock으로 상태 관리

### 5.2 상태 저장 구조

```
/var/lib/basphere/ipam/
├── allocations.tsv      # 사용자 → 블록 시작 IP
├── leases.tsv           # IP → 사용자/VM/생성시각
└── .lock                # flock용 락 파일
```

allocations.tsv 예시:
```
kimht    10.254.0.64
alicep   10.254.0.96
```

leases.tsv 예시:
```
10.254.0.70    kimht    dev-server-01    2026-01-09T19:30:00Z
10.254.0.71    kimht    dev-server-02    2026-01-09T19:35:00Z
```

### 5.3 할당 정책

- 사용자 계정 생성 시: allocate-block 자동 실행, 빈 32-IP 블록 배정
- VM 생성 시: allocate-ip 실행, 사용자 블록 내 첫 빈 IP 선택
- 예약: Gateway 10.254.0.1 제외
- VM에 주입하는 실제 마스크: /21, gateway 10.254.0.1

---

## 6. VM 프로비저닝 (Terraform + vSphere)

### 6.1 설계 원칙

- 선언적 관리: Terraform으로 VM 상태 관리
- 사용자 격리: 사용자별 독립 tfstate
- 래핑: 사용자는 Terraform 직접 사용 안함

### 6.2 Terraform 디렉토리 구조

```
/var/lib/basphere/terraform/{user}/{vm-name}/
├── main.tf              # 자동 생성된 Terraform 설정
├── terraform.tfstate    # 상태 파일
└── terraform.tfstate.backup
```

### 6.3 VM 생성 흐름

1. 사용자가 create-vm 실행
2. 래퍼가 IP 자동 할당 (allocate-ip)
3. Terraform 설정 파일 자동 생성
4. cloud-init network-config 생성 및 guestinfo 주입
5. terraform apply 실행
6. 결과 출력 (IP, VM 이름)

### 6.4 사용자 인터페이스

```bash
$ create-vm
? VM 이름: my-dev-server
? 스펙 선택:
  [1] small   - 2 vCPU, 4GB RAM, 50GB Disk
  [2] medium  - 4 vCPU, 8GB RAM, 100GB Disk
  [3] large   - 8 vCPU, 16GB RAM, 200GB Disk
? 선택: 2
? 대수: 2

✓ IP 할당: 10.254.0.70, 10.254.0.71
✓ VM 생성 중...
✓ 완료: my-dev-server-1 (10.254.0.70)
✓ 완료: my-dev-server-2 (10.254.0.71)
```

---

## 7. Kubernetes 클러스터 프로비저닝 (Cluster API)

### 7.1 설계 원칙

- Cluster API (CAPV): vSphere에서 K8s 클러스터 선언적 관리
- Management 클러스터: CAPI 컨트롤러 실행 환경
- Ubuntu OVA: Kubernetes SIG 제공 이미지 사용

### 7.2 클러스터 메타데이터 저장

```
/var/lib/basphere/clusters/{user}/{cluster-name}/
├── cluster.yaml         # CAPI manifest
├── kubeconfig           # 생성된 클러스터의 kubeconfig
└── metadata.json        # 클러스터 메타정보
```

### 7.3 클러스터 생성 흐름

1. 사용자가 create-cluster 실행
2. 래퍼가 IP 자동 할당 (Control Plane + Worker 수만큼)
3. CAPI manifest 자동 생성
4. kubectl apply -f cluster.yaml 실행 (Management 클러스터)
5. 클러스터 생성 완료 대기
6. kubeconfig 추출 및 저장
7. 결과 출력

### 7.4 사용자 인터페이스

```bash
$ create-cluster
? 클러스터 이름: my-cluster
? 클러스터 타입:
  [1] dev        - 1 Control Plane, 2 Workers
  [2] standard   - 3 Control Plane, 3 Workers
? 선택: 1
? Worker 노드 스펙:
  [1] small   - 2 vCPU, 4GB RAM
  [2] medium  - 4 vCPU, 8GB RAM
  [3] large   - 8 vCPU, 16GB RAM
? 선택: 2

✓ IP 할당: 10.254.0.72 ~ 10.254.0.74
✓ 클러스터 생성 요청 완료
✓ 프로비저닝 중...

진행 상황 확인: watch-cluster my-cluster
```

---

## 8. 사용자 사용 시나리오

### 8.1 온보딩 (승인 전)

1. 로컬에서 SSH 키 생성
   ```bash
   ssh-keygen -t ed25519
   ```
2. 공개키 제출
   ```bash
   curl -X POST https://bastion.company/key-submit \
     -d "user=kimht" \
     -d "pubkey=$(cat ~/.ssh/id_ed25519.pub)"
   ```
3. 관리자 승인 후 계정 생성 (메일 알림)

### 8.2 정상 사용 (승인 후)

1. Bastion 접속
   ```bash
   ssh kimht@bastion.company
   ```

2. 개별 VM 생성/관리
   ```bash
   create-vm              # VM 생성 (대화형)
   list-vms               # 내 VM 목록
   delete-vm <vm-name>    # VM 삭제
   ```

3. Kubernetes 클러스터 생성/관리
   ```bash
   create-cluster             # 클러스터 생성 (대화형)
   list-clusters              # 내 클러스터 목록
   get-kubeconfig <name>      # kubeconfig 출력
   watch-cluster <name>       # 프로비저닝 상태 확인
   delete-cluster <name>      # 클러스터 삭제
   ```

4. 리소스 조회
   ```bash
   list-resources         # 전체 리소스 (VM + 클러스터)
   show-quota             # 내 할당량/사용량
   ```

---

## 9. 권한 및 보안 모델

### 9.1 사용자 권한

- 사용자는 정해진 래퍼 명령만 실행 가능
- sudoers로 최소 권한 부여

```
# /etc/sudoers.d/basphere-users
%basphere-users ALL=(basphere) NOPASSWD: /usr/local/bin/basphere-*
```

### 9.2 제한 사항

- Terraform CLI 직접 사용 금지
- kubectl 직접 사용 금지 (Management 클러스터)
- vSphere 직접 접근 금지

### 9.3 감사 로깅

- 모든 래퍼 명령 실행 로그 기록
- 로그 경로: /var/log/basphere/audit.log
- 기록 항목: timestamp, user, command, parameters, result

---

## 10. 디렉토리 구조

### 10.1 Bastion 서버 디렉토리

```
/var/lib/basphere/
├── ipam/
│   ├── allocations.tsv
│   ├── leases.tsv
│   └── .lock
├── terraform/
│   └── {user}/
│       └── {vm-name}/
│           ├── main.tf
│           └── terraform.tfstate
├── clusters/
│   └── {user}/
│       └── {cluster-name}/
│           ├── cluster.yaml
│           ├── kubeconfig
│           └── metadata.json
├── users/
│   └── {user}/
│       └── metadata.json
├── key-requests/
│   └── {user}-{timestamp}.pub
└── templates/
    ├── terraform/
    │   └── vm.tf.tmpl
    └── capi/
        └── cluster.yaml.tmpl

/var/log/basphere/
├── audit.log
└── error.log

/usr/local/bin/
├── basphere-admin         # 관리자용 CLI
├── create-vm              # 사용자용: VM 생성
├── delete-vm              # 사용자용: VM 삭제
├── list-vms               # 사용자용: VM 목록
├── create-cluster         # 사용자용: 클러스터 생성
├── delete-cluster         # 사용자용: 클러스터 삭제
├── list-clusters          # 사용자용: 클러스터 목록
├── get-kubeconfig         # 사용자용: kubeconfig 조회
├── watch-cluster          # 사용자용: 클러스터 상태 확인
├── list-resources         # 사용자용: 전체 리소스 목록
└── show-quota             # 사용자용: 할당량 조회

/etc/basphere/
├── config.yaml            # 전역 설정
├── vsphere.env            # vSphere 인증 정보 (root만 읽기)
└── specs.yaml             # VM/클러스터 스펙 정의
```

### 10.2 설정 파일 예시

/etc/basphere/config.yaml:
```yaml
vsphere:
  server: vcenter.company.local
  datacenter: DC1
  datastore: datastore1
  network: dev-network
  resource_pool: /DC1/host/Cluster/Resources

templates:
  vm: ubuntu-22.04-cloud-init
  kubernetes: ubuntu-2204-kube-v1.28.0

network:
  cidr: 10.254.0.0/21
  gateway: 10.254.0.1
  dns:
    - 8.8.8.8
    - 1.1.1.1
  block_size: 32

management_cluster:
  kubeconfig: /etc/basphere/management-kubeconfig
  namespace_prefix: user-

quotas:
  default:
    max_vms: 10
    max_clusters: 3
    max_ips: 32
```

/etc/basphere/specs.yaml:
```yaml
vm_specs:
  small:
    cpu: 2
    memory: 4096
    disk: 50
  medium:
    cpu: 4
    memory: 8192
    disk: 100
  large:
    cpu: 8
    memory: 16384
    disk: 200

cluster_specs:
  dev:
    control_plane_count: 1
    worker_count: 2
    control_plane_spec: medium
    worker_spec: medium
  standard:
    control_plane_count: 3
    worker_count: 3
    control_plane_spec: medium
    worker_spec: large
```

---

## 11. 구현 산출물 (Deliverables)

### Phase 1: 기반 구축
1. 디렉토리 구조 생성 스크립트
2. 설정 파일 템플릿 (config.yaml, specs.yaml)
3. 공개키 제출 API (간단한 HTTP 서버)

### Phase 2: 사용자 관리
4. basphere-admin user add - 사용자 계정 생성
5. basphere-admin user list - 사용자 목록
6. basphere-admin user delete - 사용자 삭제

### Phase 3: IPAM
7. allocate-block - IP 블록 할당 (내부용)
8. allocate-ip - 개별 IP 할당 (내부용)
9. release-ip - IP 반환 (내부용)

### Phase 4: VM 프로비저닝
10. create-vm - VM 생성 (사용자용)
11. delete-vm - VM 삭제 (사용자용)
12. list-vms - VM 목록 (사용자용)
13. Terraform 템플릿 (vm.tf.tmpl)

### Phase 5: Kubernetes 클러스터
14. create-cluster - 클러스터 생성 (사용자용)
15. delete-cluster - 클러스터 삭제 (사용자용)
16. list-clusters - 클러스터 목록 (사용자용)
17. get-kubeconfig - kubeconfig 조회 (사용자용)
18. watch-cluster - 상태 확인 (사용자용)
19. CAPI manifest 템플릿 (cluster.yaml.tmpl)

### Phase 6: 운영 도구
20. list-resources - 전체 리소스 목록
21. show-quota - 할당량 조회
22. 감사 로깅 시스템

### Phase 7: 문서
23. 사용자 가이드
24. 관리자 가이드
25. 설치 가이드

---

## 12. 구현 우선순위 및 로드맵

### Stage 1: 최소 기능 (MVP)
목표: 사용자가 VM을 만들 수 있다

```
[1] 디렉토리 구조 + 설정 파일
[2] basphere-admin user add (수동 IP 블록 할당 포함)
[3] IPAM (allocate-block, allocate-ip)
[4] create-vm + Terraform 템플릿
[5] list-vms, delete-vm
```

### Stage 2: 클러스터 지원
목표: 사용자가 K8s 클러스터를 만들 수 있다

```
[6] CAPI manifest 템플릿
[7] create-cluster
[8] get-kubeconfig, watch-cluster
[9] list-clusters, delete-cluster
```

### Stage 3: 운영 안정화
목표: 운영 가능한 수준의 완성도

```
[10] 공개키 제출 API
[11] show-quota, list-resources
[12] 감사 로깅
[13] 문서 작성
```

---

## 13. 의도적으로 제외한 범위 (현재 단계)

- 사용자별 VLAN 자동 생성
- OPNsense VLAN/게이트웨이 자동화
- vSphere PortGroup 자동 생성
- DB 기반 IPAM / API
- Web UI / 포털 (Backstage)
- OCI 클라우드 연동
- 멀티 클라우드 지원

> 위 항목은 IDP 단계에서 다룬다.

---

## 14. 기술 스택 요약

| 구분 | 기술 |
|------|------|
| VM 프로비저닝 | Terraform + vSphere Provider |
| K8s 클러스터 | Cluster API + CAPV |
| K8s 노드 이미지 | Ubuntu OVA (Kubernetes SIG) |
| 가상화 | VMware vSphere / vCenter |
| IP 관리 | 경량 IPAM (TSV + flock) |
| 사용자 인증 | SSH 키 기반 (BYOK) |
| CLI 구현 | Bash 또는 Go |
| 설정 관리 | YAML |
