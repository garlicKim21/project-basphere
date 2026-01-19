# Basphere - Self-Service Infrastructure Platform

## 프로젝트 개요

Basphere는 VMware vSphere 기반의 셀프서비스 인프라 플랫폼입니다.
개발자가 Bastion 서버에 SSH 접속하여 직접 VM을 생성/관리할 수 있습니다.

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
                             └──────┬──────┘
                                    │
                                    ▼
                             vsphere.env
                             (600, root만 읽기)
```

**보안 아키텍처**: CLI는 API 서버를 통해 VM 작업을 수행하며, vSphere 인증 정보는 root만 접근 가능

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

### Bastion 서버
| 항목 | 값 |
|------|-----|
| Bastion IP | 172.20.0.10 |
| 관리자 계정 | basphere |
| Git 저장소 | /opt/basphere/ |
| CLI 소스 | /opt/basphere/basphere-cli/ |
| API 소스 | /opt/basphere/basphere-api/ |
| 설치된 CLI | /usr/local/bin/ (basphere-admin, create-vm 등) |
| 설치된 라이브러리 | /usr/local/lib/basphere/ |

### vSphere 환경
| 항목 | 값 |
|------|-----|
| vCenter | vcsa.basphere.local |
| Datacenter | Basphere |
| Cluster | Basphere-Home |
| Datastore | 01-VM-Block |
| VM Network | 99-basphere-cli |
| VM Folder | basphere-cli |
| Ubuntu 템플릿 | ubuntu-noble-24.04-cloudimg |
| Rocky 템플릿 | rocky-10-template |

### 네트워크 설정
| 항목 | 값 |
|------|-----|
| CIDR | 10.254.0.0/21 |
| Gateway | 10.254.0.1 |
| Netmask | 255.255.248.0 (/21) |
| MTU | 1450 (오버레이 네트워크) |
| DNS | 8.8.8.8, 1.1.1.1 |
| 사용자당 IP 블록 | /27 (32개) |

### 사용자 할당량
| 항목 | 기본값 |
|------|--------|
| 최대 VM 수 | 10 |
| 최대 클러스터 | 3 |
| 최대 IP | 32 |

## 주요 설정 파일 (Bastion)

- `/etc/basphere/config.yaml` - 메인 설정
- `/etc/basphere/vsphere.env` - vSphere 인증 정보 **(600 권한, root만 읽기)**
- `/etc/basphere/api.yaml` - API 서버 설정
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
# 코드 업데이트 및 CLI 재설치
cd /opt/basphere && sudo git pull
cd /opt/basphere/basphere-cli && sudo ./install.sh

# API 서버 빌드 및 실행 (개발 모드)
cd /opt/basphere/basphere-api && make tidy && make build-linux
sudo ./build/basphere-api-linux-amd64 --dev

# 사용자 관리
sudo basphere-admin user list
sudo basphere-admin user pending
sudo basphere-admin user approve <username>

# VM 테스트 (사용자로)
create-vm -n test -s small
list-vms
delete-vm test
```

## MVP 완료 (2026-01-19)

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
- [x] **API 기반 VM 관리** - CLI → API → Terraform 아키텍처
- [x] **vSphere 인증 정보 보호** - vsphere.env 600 권한

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

## 다음 계획

- [ ] IDP 구축
- [ ] Kubernetes 클러스터 프로비저닝 (Stage 2)
- [ ] 웹 대시보드

## 주의사항

- vSphere customization과 cloud-init을 함께 사용 시 네트워크 설정 충돌 주의
- snap으로 설치된 yq는 /etc 접근 불가 (바이너리 버전 사용)
- Terraform 상태 파일은 로컬 저장 (각 사용자별 디렉토리)
- **Ubuntu 24.04 cloud-init 네트워크 설정**: 네트워크 설정은 `guestinfo.metadata` 안에 `network` 키로 포함해야 함. 별도의 `guestinfo.network`는 작동하지 않음 (vm.tf.tmpl 참조)

## 새 환경에 설치하기

다른 환경(회사 등)에 Basphere를 이식할 때 참고하세요.

### 1. Bastion 서버 준비

```bash
# Ubuntu 22.04 LTS 권장
# 필수 사양: 2 vCPU, 4GB RAM, 50GB 디스크

# 관리자 계정 생성
sudo useradd -m -s /bin/bash basphere
sudo passwd basphere

# Git 저장소 클론
sudo mkdir -p /opt/basphere
sudo chown basphere:basphere /opt/basphere
cd /opt/basphere
git clone https://github.com/your-org/project-basphere.git .
```

### 2. 환경별 설정 파일 수정

#### /etc/basphere/config.yaml
```yaml
# 환경에 맞게 수정 필요한 항목
vsphere:
  server: "vcenter.your-company.local"    # vCenter 주소
  datacenter: "Your-DC"                   # 데이터센터 이름
  cluster: "Your-Cluster"                 # 클러스터 이름
  datastore: "Your-Datastore"             # 데이터스토어 이름
  network: "VM-Network"                   # VM 포트그룹 이름
  folder: "basphere-vms"                  # VM 폴더 이름

# OS별 템플릿 설정
# interface: OS에서 인식하는 네트워크 인터페이스 이름 (VMXNET3 기준)
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
  cidr: "10.254.0.0/21"                   # 사용자 VM용 IP 대역
  gateway: "10.254.0.1"                   # 게이트웨이
  netmask: "255.255.248.0"
  prefix_length: 21
  mtu: 1500                               # 일반 네트워크는 1500, 오버레이는 1450
  dns:
    - "8.8.8.8"
    - "1.1.1.1"
  block_size: 32                          # 사용자당 IP 개수
```

#### /etc/basphere/vsphere.env
```bash
# vCenter 인증 정보 (민감 정보!)
export VSPHERE_USER='administrator@your-domain.local'
export VSPHERE_PASSWORD='your-password'
export VSPHERE_ALLOW_UNVERIFIED_SSL='true'
```

**보안 설정 (필수)**:
```bash
sudo chmod 600 /etc/basphere/vsphere.env
sudo chown root:root /etc/basphere/vsphere.env
```

### 3. Go 설치 (API 서버 빌드용)

```bash
# Go 1.21+ 설치
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# PATH 설정 (~/.bashrc 또는 ~/.profile에 추가)
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 설치 확인
go version
```

### 4. 설치 실행

```bash
# CLI 설치 (의존성 자동 설치: jq, yq, terraform)
cd /opt/basphere/basphere-cli
sudo ./install.sh

# API 서버 빌드
cd /opt/basphere/basphere-api
make tidy && make build-linux
```

### 5. VM 템플릿 준비

vCenter에 다음 조건을 만족하는 VM 템플릿이 필요합니다.

#### Ubuntu 템플릿 (Cloud Image 사용)
```bash
# Ubuntu Cloud Image 다운로드 (OVA 형식)
# https://cloud-images.ubuntu.com/noble/current/
# noble-server-cloudimg-amd64.ova 다운로드 후 vSphere에 배포
```

#### Rocky Linux 템플릿 (ISO 설치)
```bash
# 1. Rocky Linux ISO로 VM 생성 및 설치
#    - 파티션: Standard (LVM 사용 안 함) - growpart 자동 확장을 위해
#    - 네트워크 어댑터: VMXNET3

# 2. 필수 패키지 설치
sudo dnf install -y cloud-init open-vm-tools cloud-utils-growpart
sudo systemctl enable cloud-init cloud-init-local cloud-config cloud-final vmtoolsd

# 3. 템플릿 준비 (sysprep)
sudo truncate -s 0 /etc/machine-id
sudo rm -f /etc/ssh/ssh_host_*
sudo cloud-init clean
sudo passwd -l root
# 설치 시 만든 임시 사용자 삭제
sudo userdel -r <임시사용자>
history -c
sudo shutdown -h now

# 4. vSphere에서 VM을 템플릿으로 변환
```

#### 템플릿 요구사항
- cloud-init 설치 및 활성화
- open-vm-tools 설치
- cloud-utils-growpart 설치 (디스크 자동 확장)
- 네트워크 어댑터: VMXNET3
- 파티션: Standard (LVM 미사용 권장)

### 6. 환경별 체크리스트

| 항목 | 홈랩 | 회사 |
|------|------|------|
| Bastion IP | 172.20.0.10 | 환경에 맞게 |
| vCenter | vcsa.basphere.local | 회사 vCenter |
| Datacenter | Basphere | 회사 DC |
| 네트워크 대역 | 10.254.0.0/21 | 할당받은 대역 |
| MTU | 1450 (오버레이) | 1500 (일반) |
| DNS | 8.8.8.8 | 회사 DNS |

### 7. API 서버 설정

```bash
# systemd 서비스 등록
sudo cp /opt/basphere/basphere-api/basphere-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable basphere-api
sudo systemctl start basphere-api
```

### 8. 설치 후 확인

```bash
# CLI 동작 확인
sudo basphere-admin user list

# API 서버 상태 확인
sudo systemctl status basphere-api

# 헬스 체크
curl http://localhost:8080/health

# 웹 폼 접속 테스트
curl http://localhost:8080/register
```
