# Basphere CLI

Bastion 기반 셀프서비스 VM/Kubernetes 프로비저닝 CLI 도구

## 개요

IDP(포털) 구축 전 단계에서 개발자가 Bastion 서버에 SSH 접속하여 CLI를 통해 VMware vSphere 상에 VM을 생성하고 관리할 수 있도록 하는 도구입니다.

### 아키텍처

```
┌─────────────┐     SSH      ┌─────────────┐   Terraform   ┌─────────────┐
│  Developer  │ ──────────▶  │   Bastion   │ ───────────▶  │   vCenter   │
│  Workstation│              │   Server    │               │   (vSphere) │
└─────────────┘              └─────────────┘               └─────────────┘
                                   │
                                   │ IPAM, User Management
                                   ▼
                             ┌─────────────┐
                             │  /var/lib/  │
                             │  basphere/  │
                             └─────────────┘
```

### 기능

| Stage | 기능 | 상태 |
|-------|------|------|
| Stage 1 | 사용자 계정 관리 | ✅ 완료 |
| Stage 1 | IP 자동 할당 (경량 IPAM) | ✅ 완료 |
| Stage 1 | VM 생성/조회/삭제 (Terraform) | ✅ 완료 |
| Stage 2 | Kubernetes 클러스터 생성 (Cluster API) | 🚧 예정 |

---

## 설치 가이드 (운영자용)

### 1. 사전 요구사항

#### Bastion 서버
- Ubuntu 22.04 LTS 권장
- 인터넷 접근 가능 (Terraform provider 다운로드)
- vCenter 네트워크 접근 가능

#### 필수 패키지
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y jq git curl

# yq 설치
sudo wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
sudo chmod +x /usr/local/bin/yq

# Terraform 설치 (1.0+)
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform
```

#### vSphere 환경
- vCenter 6.7+ 또는 vSphere 7.0+
- VM 템플릿: Ubuntu Cloud Image OVA
  - 다운로드: https://cloud-images.ubuntu.com/jammy/current/
  - 파일: `jammy-server-cloudimg-amd64.ova`
- VM이 배치될 폴더 (예: `basphere-vms`)
- 네트워크 (포트그룹)
- 데이터스토어

### 2. VM 템플릿 준비

vCenter에서 Ubuntu Cloud Image OVA를 템플릿으로 등록:

1. vCenter Web Client 접속
2. **Actions** → **Deploy OVF Template**
3. OVA 파일 선택 및 배포
4. 배포된 VM을 **템플릿으로 변환** (우클릭 → Convert to Template)
5. 템플릿 이름 기록 (예: `ubuntu-jammy-22.04-cloudimg`)

### 3. Basphere CLI 설치

```bash
# 저장소 클론
git clone <repository-url>
cd basphere-cli

# 설치 (root 권한 필요)
sudo ./install.sh
```

설치 스크립트가 수행하는 작업:
- 필수 패키지 확인 및 설치
- 디렉토리 생성 (`/var/lib/basphere`, `/etc/basphere`, `/var/log/basphere`)
- 그룹 생성 (`basphere-users`, `basphere-admin`)
- 서비스 계정 생성 (`basphere`)
- IPAM 초기화
- CLI 스크립트 설치
- 권한 설정
- sudoers 설정

### 4. 설정 파일 수정

#### vSphere 연결 설정
```bash
sudo vim /etc/basphere/config.yaml
```

```yaml
vsphere:
  server: "vcenter.your-domain.local"    # vCenter 주소
  datacenter: "Datacenter"               # 데이터센터 이름
  cluster: "Cluster"                     # 클러스터 이름
  datastore: "datastore1"                # 데이터스토어 이름
  network: "VM Network"                  # 포트그룹 이름
  resource_pool: ""                      # 리소스풀 (비워두면 클러스터 기본값)
  folder: "basphere-vms"                 # VM 폴더

templates:
  vm: "ubuntu-jammy-22.04-cloudimg"      # VM 템플릿 이름

network:
  cidr: "10.254.0.0/21"                  # VM에 할당할 IP 대역
  gateway: "10.254.0.1"                  # 게이트웨이
  dns:
    - "8.8.8.8"
    - "1.1.1.1"
  netmask: "255.255.248.0"               # 서브넷 마스크
  prefix_length: 21                       # CIDR prefix
  block_size: 32                          # 사용자당 IP 개수

quotas:
  default:
    max_vms: 10                           # 사용자당 최대 VM
    max_clusters: 3                       # 사용자당 최대 클러스터
    max_ips: 32                           # 사용자당 최대 IP
```

#### vSphere 인증 정보
```bash
sudo vim /etc/basphere/vsphere.env
```

```bash
VSPHERE_USER="administrator@vsphere.local"
VSPHERE_PASSWORD="your-password"
```

#### VM 스펙 정의 (선택)
```bash
sudo vim /etc/basphere/specs.yaml
```

```yaml
vm_specs:
  small:
    cpu: 2
    memory_mb: 4096
    disk_gb: 50
  medium:
    cpu: 4
    memory_mb: 8192
    disk_gb: 100
  large:
    cpu: 8
    memory_mb: 16384
    disk_gb: 200
```

### 5. 관리자 설정

```bash
# 현재 사용자를 basphere-admin 그룹에 추가
sudo usermod -aG basphere-admin $(whoami)

# 로그아웃 후 다시 로그인하여 그룹 적용
exit
```

### 6. 설치 확인

```bash
# 관리자 CLI 확인
sudo basphere-admin --help

# 사용자 CLI 확인 (경로)
which create-vm list-vms delete-vm show-quota list-resources
```

---

## 사용자 관리

### 사용자 추가

```bash
# 사용자의 SSH 공개키 파일이 필요
sudo basphere-admin user add <username> --pubkey /path/to/id_ed25519.pub
```

이 명령은:
1. Linux 시스템 사용자 생성
2. SSH 키 설정 (`~/.ssh/authorized_keys`)
3. `basphere-users` 그룹에 추가
4. IP 블록 자동 할당
5. 사용자 데이터 디렉토리 생성

### 사용자 목록

```bash
sudo basphere-admin user list
```

### 사용자 정보 조회

```bash
sudo basphere-admin user show <username>
```

### 사용자 삭제

```bash
# VM이 있으면 먼저 삭제해야 함
sudo basphere-admin user delete <username>
```

---

## 디렉토리 구조

```
basphere-cli/
├── install.sh                    # 설치 스크립트
├── README.md                     # 운영자 가이드 (이 문서)
├── docs/
│   └── user-guide.md             # 사용자 가이드
├── config/
│   ├── config.yaml.example       # 전역 설정 템플릿
│   ├── specs.yaml.example        # VM 스펙 정의
│   └── vsphere.env.example       # vSphere 인증 정보
├── lib/
│   └── common.sh                 # 공통 함수 라이브러리
├── scripts/
│   ├── basphere-admin            # 관리자 CLI
│   ├── internal/                 # 내부 스크립트 (IPAM 등)
│   │   ├── ipam-common.sh
│   │   ├── allocate-block
│   │   ├── allocate-ip
│   │   ├── release-ip
│   │   └── list-user-ips
│   └── user/                     # 사용자 CLI
│       ├── create-vm
│       ├── delete-vm
│       ├── list-vms
│       ├── list-resources
│       └── show-quota
└── templates/
    └── terraform/
        ├── vm.tf.tmpl            # Terraform VM 템플릿
        └── user-folder.tf.tmpl   # 사용자 폴더 템플릿
```

### 설치 후 디렉토리

```
/etc/basphere/                    # 설정 파일
├── config.yaml
├── specs.yaml
└── vsphere.env

/var/lib/basphere/                # 데이터
├── ipam/                         # IP 할당 정보
│   ├── allocations.tsv           # 사용자별 IP 블록
│   └── leases.tsv                # 개별 IP 할당
├── users/                        # 사용자 메타데이터
│   └── <username>/
│       └── metadata.json
├── terraform/                    # Terraform 상태
│   └── <username>/
│       ├── _folder/              # vSphere 사용자 폴더 Terraform
│       │   ├── main.tf
│       │   └── terraform.tfstate
│       └── <vm-name>/
│           ├── main.tf
│           ├── metadata.json
│           └── terraform.tfstate
├── clusters/                     # 클러스터 데이터 (Stage 2)
└── templates/                    # 템플릿 파일
    └── terraform/
        ├── vm.tf.tmpl
        └── user-folder.tf.tmpl

/var/log/basphere/                # 로그
└── audit.log                     # 감사 로그

/usr/local/bin/                   # CLI 명령어
├── basphere-admin
├── create-vm
├── delete-vm
├── list-vms
├── list-resources
└── show-quota

/usr/local/lib/basphere/          # 라이브러리
├── common.sh
└── internal/
    └── (IPAM 스크립트들)
```

---

## 트러블슈팅

### Permission denied 오류

권한 문제 발생 시:
```bash
# 권한 재설정
sudo /path/to/basphere-cli/install.sh
```

또는 수동으로:
```bash
sudo chmod 755 /var/lib/basphere /var/lib/basphere/users /var/lib/basphere/terraform
sudo chmod 777 /var/lib/basphere/ipam /var/log/basphere
sudo chmod 666 /var/lib/basphere/ipam/.lock /var/lib/basphere/ipam/leases.tsv
sudo chmod 644 /var/lib/basphere/ipam/allocations.tsv
sudo chmod 644 /etc/basphere/config.yaml /etc/basphere/specs.yaml /etc/basphere/vsphere.env
```

### Terraform 오류

#### "network not found"
- `config.yaml`의 `network` 값이 vCenter의 포트그룹 이름과 일치하는지 확인

#### "template not found"
- `config.yaml`의 `templates.vm` 값이 vCenter의 템플릿 이름과 일치하는지 확인
- 템플릿이 지정된 데이터센터에 있는지 확인

#### "CDROM device required"
- VM 템플릿이 vApp 속성을 사용하는 경우 발생
- `vm.tf.tmpl`에 `cdrom { client_device = true }` 블록 확인

### IP 할당 실패

```bash
# IP 블록 확인
cat /var/lib/basphere/ipam/allocations.tsv

# IP 사용 현황 확인
cat /var/lib/basphere/ipam/leases.tsv

# 수동 IP 블록 할당
sudo /usr/local/lib/basphere/internal/allocate-block <username>
```

### 로그 확인

```bash
# 감사 로그
cat /var/log/basphere/audit.log

# Terraform 로그 (VM별)
cat /var/lib/basphere/terraform/<username>/<vm-name>/terraform-apply.log
```

---

## 네트워크 설계

기본 설정:
- 전체 대역: `10.254.0.0/21` (2048개 IP)
- 사용자당: `/27` 블록 (32개 IP)
- 최대 사용자: 약 60명

IP 블록 할당 예시:
| 사용자 | IP 블록 | 범위 |
|--------|---------|------|
| user1 | 10.254.0.32/27 | 10.254.0.32 - 10.254.0.63 |
| user2 | 10.254.0.64/27 | 10.254.0.64 - 10.254.0.95 |
| user3 | 10.254.0.96/27 | 10.254.0.96 - 10.254.0.127 |

---

## vSphere 구조

### 폴더 구조

vSphere에서 VM은 사용자별 폴더로 구성됩니다:

```
basphere-vms/                     # 기본 폴더 (config.yaml의 vsphere.folder)
├── user1/                        # 사용자 폴더
│   ├── user1-web-server          # VM (사용자 프리픽스 포함)
│   └── user1-db-server
├── user2/
│   ├── user2-app-1
│   └── user2-app-2
└── ...
```

### VM 이름 규칙

- **CLI에서 사용하는 이름**: 사용자가 지정한 짧은 이름 (예: `web-server`)
- **vSphere에서의 이름**: 사용자 프리픽스 + 이름 (예: `user1-web-server`)

이렇게 하면:
- 사용자별 VM을 vCenter에서 쉽게 구분
- vSphere 전체에서 VM 이름 고유성 보장
- CLI에서는 짧은 이름으로 편리하게 사용

### 폴더 생성 시점

- 사용자 폴더는 `basphere-admin user add` 명령 시 자동 생성
- 사용자 삭제 시 폴더도 함께 삭제 (VM이 없는 경우만)

---

## 라이선스

MIT License
