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
| MVP | 사용자 계정 관리 | ✅ 완료 |
| MVP | 웹 기반 사용자 등록 요청 | ✅ 완료 |
| MVP | 관리자 승인/거부 | ✅ 완료 |
| MVP | IP 자동 할당 (경량 IPAM) | ✅ 완료 |
| MVP | VM 생성/조회/삭제 (Terraform) | ✅ 완료 |
| MVP | 다중 OS 지원 (Ubuntu, Rocky Linux) | ✅ 완료 |
| MVP | 디스크 자동 확장 | ✅ 완료 |
| Stage 2 | Kubernetes 클러스터 생성 (Cluster API) | 🚧 예정 |

### 지원 OS

| OS | 설명 | 네트워크 인터페이스 |
|----|------|-------------------|
| Ubuntu 24.04 LTS | Ubuntu Cloud Image 기반 | ens192 |
| Rocky Linux 10.1 | ISO 설치 기반 | ens33 |

### VM 스펙

| 스펙 | vCPU | RAM | Disk | 용도 |
|------|------|-----|------|------|
| tiny | 2 | 4GB | 50GB | 테스트용 |
| small | 2 | 8GB | 50GB | 개발용 |
| medium | 4 | 16GB | 100GB | 일반 워크로드 |
| large | 8 | 32GB | 200GB | 고성능 워크로드 |
| huge | 16 | 64GB | 200GB | 대규모 워크로드 |

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

# yq 설치 (반드시 바이너리로 설치 - snap 버전은 /etc 접근 불가)
sudo wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
sudo chmod +x /usr/local/bin/yq
# 주의: snap install yq 사용 금지 (샌드박스로 인해 /etc/basphere 접근 불가)

# Terraform 설치 (1.0+)
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform
```

#### vSphere 환경
- vCenter 6.7+ 또는 vSphere 7.0+
- VM 템플릿: cloud-init 지원 필수
- VM이 배치될 폴더 (예: `basphere-vms`)
- 네트워크 (포트그룹)
- 데이터스토어

### 2. VM 템플릿 준비

#### Ubuntu 템플릿 (Cloud Image 사용)

1. Ubuntu Cloud Image 다운로드
   - URL: https://cloud-images.ubuntu.com/noble/current/
   - 파일: `noble-server-cloudimg-amd64.ova`

2. vCenter에서 OVA 배포
   - **Actions** → **Deploy OVF Template**
   - OVA 파일 선택 및 배포
   - 네트워크 어댑터: **VMXNET3** 확인

3. 템플릿으로 변환
   - 배포된 VM 우클릭 → **Convert to Template**
   - 템플릿 이름: `ubuntu-noble-24.04-cloudimg`

#### Rocky Linux 템플릿 (ISO 설치)

1. Rocky Linux ISO로 VM 생성 및 설치
   - 파티션: **Standard** (LVM 사용 안 함) - growpart 자동 확장을 위해
   - 네트워크 어댑터: **VMXNET3**

2. 필수 패키지 설치
   ```bash
   sudo dnf install -y cloud-init open-vm-tools cloud-utils-growpart
   sudo systemctl enable cloud-init cloud-init-local cloud-config cloud-final vmtoolsd
   ```

3. 템플릿 준비 (sysprep)
   ```bash
   sudo truncate -s 0 /etc/machine-id
   sudo rm -f /etc/ssh/ssh_host_*
   sudo cloud-init clean
   sudo passwd -l root
   # 설치 시 만든 임시 사용자 삭제 (콘솔에서 실행)
   sudo userdel -r <임시사용자>
   history -c
   sudo shutdown -h now
   ```

4. 템플릿으로 변환
   - VM 우클릭 → **Convert to Template**
   - 템플릿 이름: `rocky-10-template`

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

# OS별 템플릿 설정
# interface: OS에서 인식하는 네트워크 인터페이스 이름
templates:
  os:
    ubuntu-24.04:
      template: "ubuntu-noble-24.04-cloudimg"
      default_user: "ubuntu"
      description: "Ubuntu 24.04 LTS (Noble)"
      interface: "ens192"
    rocky-10.1:
      template: "rocky-10-template"
      default_user: "rocky"
      description: "Rocky Linux 10.1"
      interface: "ens33"

network:
  cidr: "10.254.0.0/21"                  # VM에 할당할 IP 대역
  gateway: "10.254.0.1"                  # 게이트웨이
  dns:
    - "8.8.8.8"
    - "1.1.1.1"
  netmask: "255.255.248.0"               # 서브넷 마스크
  prefix_length: 21                       # CIDR prefix
  mtu: 1500                               # MTU (오버레이 네트워크는 1450)
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

#### VM 스펙 정의
```bash
sudo vim /etc/basphere/specs.yaml
```

```yaml
vm_specs:
  tiny:
    description: "초소형 VM (테스트용)"
    cpu: 2
    memory_mb: 4096      # 4GB
    disk_gb: 50
  small:
    description: "소형 VM (개발용)"
    cpu: 2
    memory_mb: 8192      # 8GB
    disk_gb: 50
  medium:
    description: "중형 VM (일반 워크로드)"
    cpu: 4
    memory_mb: 16384     # 16GB
    disk_gb: 100
  large:
    description: "대형 VM (고성능 워크로드)"
    cpu: 8
    memory_mb: 32768     # 32GB
    disk_gb: 200
  huge:
    description: "초대형 VM (대규모 워크로드)"
    cpu: 16
    memory_mb: 65536     # 64GB
    disk_gb: 200

defaults:
  vm_spec: "small"
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

### 사용자 등록 요청 (웹 기반)

basphere-api 서버 실행 시 웹 폼을 통한 등록 요청 가능:
```bash
# API 서버 실행
sudo /opt/basphere/basphere-api/build/basphere-api-linux-amd64 --dev

# 등록 폼: http://<bastion-ip>:8080/register
```

### 대기 중인 사용자 확인 및 승인

```bash
# 대기 목록
sudo basphere-admin user pending

# 승인
sudo basphere-admin user approve <username>

# 거부
sudo basphere-admin user reject <username>
```

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

## VM 사용 (사용자용)

### VM 생성

```bash
# 대화형 모드
create-vm

# 명령행 모드
create-vm -n my-server -o ubuntu-24.04 -s small
create-vm -n db-server -o rocky-10.1 -s medium

# 여러 대 생성
create-vm -n web -o ubuntu-24.04 -s tiny -c 3
```

### VM 목록

```bash
list-vms
list-vms -a        # 상세 정보
list-vms -j        # JSON 출력
```

### VM 삭제

```bash
delete-vm my-server
delete-vm my-server -f    # 확인 없이 삭제
```

### 리소스 확인

```bash
show-quota         # 할당량 확인
list-resources     # 리소스 목록
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

### Terraform 오류

#### "network not found"
- `config.yaml`의 `network` 값이 vCenter의 포트그룹 이름과 일치하는지 확인

#### "template not found"
- `config.yaml`의 `templates.os.<os>.template` 값이 vCenter의 템플릿 이름과 일치하는지 확인
- 템플릿이 지정된 데이터센터에 있는지 확인

### cloud-init 네트워크 설정 문제

#### Ubuntu 24.04
- 네트워크 설정은 `guestinfo.metadata` 안에 `network` 키로 포함해야 함
- 별도의 `guestinfo.network`는 작동하지 않음

#### Rocky Linux
- 네트워크 인터페이스 이름이 Ubuntu와 다름 (ens33 vs ens192)
- `config.yaml`의 `interface` 설정 확인

### 디스크 확장 안 됨

Rocky Linux 템플릿에 `cloud-utils-growpart` 설치 필요:
```bash
sudo dnf install -y cloud-utils-growpart
```

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

### yq 설정 읽기 실패

`get_config` 함수가 기본값만 반환하는 경우:

```bash
# yq가 snap으로 설치되었는지 확인
which yq
# /snap/bin/yq 로 나오면 snap 버전

# snap 버전은 /etc 디렉토리 접근 불가 (샌드박스 제한)
# 해결: snap 제거 후 바이너리로 재설치
sudo snap remove yq
sudo wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
sudo chmod +x /usr/local/bin/yq
hash -r  # 셸 캐시 초기화
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
