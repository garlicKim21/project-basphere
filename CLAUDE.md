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
├── basphere-api/           # Go 기반 REST API 서버
│   ├── cmd/basphere-api/   # 서버 진입점
│   ├── internal/
│   │   ├── handler/        # HTTP 핸들러
│   │   ├── model/          # 데이터 모델
│   │   ├── store/          # 저장소 (파일 기반, 향후 DB)
│   │   └── provisioner/    # bash 스크립트 호출
│   └── web/templates/      # HTML 템플릿 (등록 폼)
│
└── deploy/                 # 배포 설정
    ├── nginx/              # nginx 리버스 프록시 설정
    └── systemd/            # systemd 서비스 파일
```

## 기술 스택

- **CLI**: Bash, jq, yq
- **API**: Go 1.21+, chi router
- **IaC**: Terraform + vSphere Provider
- **VM 초기화**: cloud-init
- **향후 IDP**: Go 기반, PostgreSQL

## 배포 환경 (예시)

> **참고**: 실제 환경 정보는 `CLAUDE.local.md`에 있습니다 (git 제외).

### Bastion 서버
| 항목 | 값 |
|------|-----|
| Bastion IP | `<bastion-ip>` |
| 관리자 계정 | basphere |
| Git 저장소 | /opt/basphere/ |
| CLI 소스 | /opt/basphere/basphere-cli/ |
| API 소스 | /opt/basphere/basphere-api/ |
| 설치된 CLI | /usr/local/bin/ (basphere-admin, create-vm 등) |
| 설치된 라이브러리 | /usr/local/lib/basphere/ |

### vSphere 환경
| 항목 | 값 |
|------|-----|
| vCenter | `vcenter.example.com` |
| Datacenter | `Your-Datacenter` |
| Cluster | `Your-Cluster` |
| Datastore | `Your-Datastore` |
| VM Network | `VM-Network` |
| VM Folder | `basphere-vms` |
| Ubuntu 템플릿 | ubuntu-noble-24.04-cloudimg |
| Rocky 템플릿 | rocky-10-template |

### 네트워크 설정 (예시)
| 항목 | 값 |
|------|-----|
| CIDR | 10.0.0.0/21 |
| Gateway | 10.0.0.1 |
| Netmask | 255.255.248.0 (/21) |
| MTU | 1500 (일반) / 1450 (오버레이) |
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
- [x] **nginx 리버스 프록시** - 내부 API 외부 차단, 공개 엔드포인트만 노출
- [x] **Google reCAPTCHA v2** - 등록 폼 봇 방지
- [x] **SSH 키 변경 요청** - 웹 폼 + 관리자 승인 워크플로우
- [x] **이메일 도메인 검증** - 허용된 도메인만 등록 가능
- [x] **SSH 보안 강화** - 비밀번호 인증 비활성화, fail2ban
- [x] **외부 접근 지원** - bastion 주소/포트 설정 가능

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
# Ubuntu 22.04/24.04 LTS 권장
# 필수 사양: 2 vCPU, 4GB RAM, 50GB 디스크

# 관리자 계정 생성
sudo useradd -m -s /bin/bash basphere
sudo passwd basphere

# Git 저장소 클론 (/opt에서 sudo 필요)
cd /opt
sudo git clone https://github.com/your-org/project-basphere.git basphere
sudo chown -R basphere:basphere /opt/basphere
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

### 3. 빌드 도구 설치

```bash
# make 설치
sudo apt install -y make

# Go 설치 (방법 1: apt - 간편)
sudo apt install -y golang-go
go version  # 1.22+ 확인

# Go 설치 (방법 2: 공식 바이너리 - 최신 버전 필요 시)
# wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
# sudo rm -rf /usr/local/go
# sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
# echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
# source ~/.bashrc
```

### 4. 설치 실행

```bash
# CLI 설치 (의존성 자동 설치: jq, yq, terraform)
cd /opt/basphere/basphere-cli
sudo ./install.sh

# API 서버 빌드 (/opt 디렉토리이므로 sudo 필요)
cd /opt/basphere/basphere-api
sudo make tidy && sudo make build-linux
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

| 항목 | 설정 예시 | 참고 |
|------|----------|------|
| Bastion IP | `<your-bastion-ip>` | 환경에 맞게 설정 |
| vCenter | `vcenter.example.com` | vCenter 주소 |
| Datacenter | `Your-Datacenter` | vSphere 데이터센터 이름 |
| 네트워크 대역 | `10.0.0.0/21` | 할당받은 대역 |
| MTU | 1500 (일반) / 1450 (오버레이) | 네트워크 환경에 따라 |
| DNS | `8.8.8.8` 또는 사내 DNS | 환경에 맞게 |

### 7. API 서버 설정

#### /etc/basphere/api.yaml
```yaml
# Basphere API Server Configuration

server:
  host: "127.0.0.1"        # nginx 뒤에서 실행 (로컬만 바인딩)
  port: 8080

storage:
  # 대기 중인 등록 요청 저장 디렉토리
  pending_dir: "/var/lib/basphere/pending"

provisioner:
  # basphere-admin 스크립트 경로
  admin_script: "/usr/local/bin/basphere-admin"

recaptcha:
  enabled: true                                    # false로 설정 시 reCAPTCHA 비활성화
  site_key: "your-recaptcha-site-key"              # Google reCAPTCHA v2 사이트 키
  secret_key: "your-recaptcha-secret-key"          # Google reCAPTCHA v2 시크릿 키

validation:
  # 허용된 이메일 도메인 (빈 배열 = 모든 도메인 허용)
  allowed_email_domains: []
  # 예: allowed_email_domains: ["company.com", "corp.company.com"]

bastion:
  # 등록 완료 페이지에 표시할 Bastion 서버 주소
  address: "bastion.example.com"                   # 또는 IP 주소
  # SSH 포트 (기본: 22, 외부 노출 시 비표준 포트 사용 권장)
  port: 22                                         # 외부 노출 시 예: 50022
```

#### systemd 서비스 등록
```bash
sudo cp /opt/basphere/basphere-api/basphere-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable basphere-api
sudo systemctl start basphere-api
```

### 8. Bastion 서버 보안 강화

#### SSH 보안 설정

`/etc/ssh/sshd_config.d/40-basphere-hardening.conf` 생성:

```bash
cat << 'EOF' | sudo tee /etc/ssh/sshd_config.d/40-basphere-hardening.conf
# Basphere SSH Hardening Configuration

# === 인증 설정 ===
PasswordAuthentication no          # 비밀번호 인증 비활성화 (SSH 키만 허용)
PubkeyAuthentication yes
PermitEmptyPasswords no
PermitRootLogin no                 # root 직접 로그인 비활성화
MaxAuthTries 3                     # 최대 인증 시도 횟수
LoginGraceTime 30                  # 로그인 유예 시간 (초)

# === 다중 세션 설정 ===
MaxSessions 20                     # 연결당 최대 세션 수 (ProxyJump, 다중 터미널)
MaxStartups 10:50:30               # 동시 미인증 연결 제한 (start:rate:full)

# === 기타 보안 설정 ===
PermitUserEnvironment no
AllowTcpForwarding yes             # ProxyJump 필요
Banner none
UseDNS no                          # DNS 역방향 조회 비활성화 (속도 향상)
EOF

# 설정 검증 및 적용
sudo sshd -t && sudo systemctl restart ssh
```

> **참고**: 파일명이 `40-`으로 시작해야 cloud-init의 `50-cloud-init.conf`보다 먼저 로드되어 설정이 적용됩니다.

#### fail2ban 설치 및 설정

```bash
# fail2ban 설치
sudo apt install -y fail2ban

# SSH jail 설정
cat << 'EOF' | sudo tee /etc/fail2ban/jail.local
# Basphere fail2ban Configuration

[DEFAULT]
bantime = 10m
findtime = 5m
maxretry = 5
banaction = iptables-multiport

[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 3                       # SSH는 3회 실패 시 차단
bantime = 30m                      # 30분 차단
findtime = 10m                     # 10분 내 실패 횟수 감시
EOF

# fail2ban 재시작
sudo systemctl restart fail2ban

# 상태 확인
sudo fail2ban-client status sshd
```

#### 보안 설정 요약

| 설정 | 값 | 설명 |
|------|-----|------|
| `PasswordAuthentication` | no | SSH 키 인증만 허용 |
| `PermitRootLogin` | no | root 로그인 차단 |
| `MaxAuthTries` | 3 | 인증 시도 제한 |
| `MaxSessions` | 20 | 다중 세션 허용 |
| `MaxStartups` | 10:50:30 | DoS 방지 |
| fail2ban `maxretry` | 3 | 3회 실패 시 차단 |
| fail2ban `bantime` | 30분 | 차단 시간 |

### 9. 외부 접근 설정 (선택사항)

Bastion 서버를 외부에서 접근 가능하게 설정할 때 참고하세요.

#### 포트 포워딩 (방화벽/라우터)

```
외부 포트 50022 → 내부 <bastion-ip>:22
```

> **주의**: 비표준 포트(50022 등)를 사용하여 자동화된 공격을 줄일 수 있습니다.

#### DNS 설정

| 레코드 | 값 | 참고 |
|--------|-----|------|
| `bastion.example.com` | A 레코드 → 공인 IP | Cloudflare 사용 시 **DNS Only** (회색 구름) |

> **Cloudflare 주의**: Proxied(주황색 구름) 모드는 HTTP/HTTPS만 지원하므로, SSH(TCP)는 **DNS Only**로 설정해야 합니다.

#### api.yaml bastion 설정 예시

```yaml
# 내부망 접속 (IP 사용)
bastion:
  address: "<bastion-ip>"
  port: 22

# 외부 도메인 접속
bastion:
  address: "bastion.example.com"
  port: 50022
```

등록 완료 페이지에 SSH 접속 명령어가 자동으로 표시됩니다:
- 포트 22: `ssh username@bastion.example.com`
- 포트 50022: `ssh -p 50022 username@bastion.example.com`

### 10. 설치 후 확인

```bash
# CLI 동작 확인
sudo basphere-admin user list

# API 서버 상태 확인
sudo systemctl status basphere-api

# 헬스 체크
curl http://localhost:8080/health

# 웹 폼 접속 테스트
curl http://localhost:8080/register

# SSH 보안 설정 확인
sudo sshd -T | grep -E "^(passwordauthentication|permitrootlogin|maxsessions)"

# fail2ban 상태 확인
sudo fail2ban-client status sshd
```

### 11. 트러블슈팅

새 환경에 배포할 때 자주 발생하는 문제와 해결 방법입니다.

#### 체크리스트

| 단계 | 확인 항목 | 확인 명령어 |
|------|----------|------------|
| 1 | Git 저장소 클론 (sudo 사용) | `ls -la /opt/basphere` |
| 2 | make 설치됨 | `which make` |
| 3 | Go 설치됨 (1.22+) | `go version` |
| 4 | CLI 설치됨 | `which basphere-admin` |
| 5 | API 빌드됨 | `ls /opt/basphere/basphere-api/build/` |
| 6 | config.yaml 설정됨 | `cat /etc/basphere/config.yaml` |
| 7 | vsphere.env 설정됨 (600 권한) | `ls -la /etc/basphere/vsphere.env` |
| 8 | api.yaml bastion 섹션 있음 | `grep -A2 "^bastion:" /etc/basphere/api.yaml` |
| 9 | vSphere 연결 성공 | `curl -k https://vcenter/rest/com/vmware/cis/session` |
| 10 | API 서버 실행 중 | `systemctl status basphere-api` |
| 11 | 웹 폼 접속 가능 | `curl http://localhost:8080/register` |
| 12 | SSH 보안 설정 적용됨 | `sudo sshd -T \| grep passwordauthentication` |
| 13 | fail2ban 실행 중 | `sudo fail2ban-client status sshd` |

#### 문제 1: 빌드 도구 없음

**증상**: `make: command not found` 또는 `go: command not found`

**원인**: Ubuntu 서버에 make, Go가 기본 설치되어 있지 않음

**해결**:
```bash
sudo apt install -y make golang-go
```

#### 문제 2: /opt에서 권한 오류

**증상**: `Permission denied` when cloning or building

**원인**: /opt 디렉토리는 root 소유이므로 일반 사용자로 작업 불가

**해결**:
```bash
# Git 클론
cd /opt
sudo git clone https://github.com/your-org/project-basphere.git basphere

# API 빌드 (sudo 필요)
cd /opt/basphere/basphere-api
sudo make tidy && sudo make build-linux
```

#### 문제 3: 등록 완료 페이지에 "bastion-server" 표시

**증상**: 등록 완료 페이지에 실제 bastion 주소 대신 "bastion-server" 표시

**원인**: api.yaml에 `bastion` 섹션이 누락됨

**해결**:
```bash
# api.yaml에 bastion 섹션 추가
sudo vi /etc/basphere/api.yaml
```

```yaml
bastion:
  address: "bastion.your-company.com"  # 실제 주소로 변경
  port: 50022                           # 실제 포트로 변경
```

```bash
# API 서버 재시작
sudo systemctl restart basphere-api
```

#### 문제 4: vSphere 인증 실패 (401 Unauthorized)

**증상**: VM 생성 시 "401 Unauthorized" 오류

**원인**: vsphere.env의 인증 정보가 잘못됨

**확인 방법**:
```bash
# vSphere API 직접 테스트
source /etc/basphere/vsphere.env
curl -k -u "${VSPHERE_USER}:${VSPHERE_PASSWORD}" \
  "https://your-vcenter/rest/com/vmware/cis/session" -X POST
```

**해결**:
```bash
# vsphere.env 수정
sudo vi /etc/basphere/vsphere.env

# 인증 정보 확인 후 수정
export VSPHERE_USER='correct-user@domain.local'
export VSPHERE_PASSWORD='correct-password'
```

#### 문제 5: 사용자 삭제 실패

**증상**: `userdel: user xxx is currently used by process yyy`

**원인**: 해당 사용자의 SSH 세션이 아직 활성화되어 있음

**해결**:
```bash
# 방법 1: 사용자 프로세스 확인 후 대기
ps -u <username>

# 세션이 종료되면 삭제
sudo userdel -r <username>

# 방법 2: 강제 종료 (주의: 사용자 작업 중단됨)
sudo pkill -u <username>
sudo userdel -r <username>
```

#### 문제 6: SSH 보안 설정이 적용되지 않음

**증상**: `PasswordAuthentication yes`가 여전히 활성화됨

**원인**: sshd_config.d 파일은 알파벳 순으로 로드되어, cloud-init의 `50-cloud-init.conf`가 나중에 로드되면 덮어씌워짐

**확인 방법**:
```bash
# 실제 적용된 설정 확인
sudo sshd -T | grep passwordauthentication

# sshd_config.d 파일 목록 확인
ls -la /etc/ssh/sshd_config.d/
```

**해결**:
```bash
# 파일명을 40-으로 변경하여 50-cloud-init.conf보다 먼저 로드
sudo mv /etc/ssh/sshd_config.d/99-basphere-hardening.conf \
        /etc/ssh/sshd_config.d/40-basphere-hardening.conf

# 또는 cloud-init 설정에서 PasswordAuthentication 제거
sudo vi /etc/ssh/sshd_config.d/50-cloud-init.conf

# SSH 재시작
sudo sshd -t && sudo systemctl restart ssh
```

#### 문제 7: sudo 비밀번호 요청

**증상**: basphere 사용자로 sudo 실행 시 비밀번호 요청

**해결**:
```bash
# basphere 사용자에게 NOPASSWD 권한 부여
echo "basphere ALL=(ALL) NOPASSWD:ALL" | sudo tee /etc/sudoers.d/basphere
sudo chmod 440 /etc/sudoers.d/basphere
```

#### 문제 8: Terraform 폴더 생성 실패

**증상**: VM 생성 시 "folder not found" 오류

**원인**: config.yaml에 지정된 VM 폴더가 vSphere에 존재하지 않음

**해결**:
```bash
# vSphere Web Client에서 폴더 생성
# 또는 govc 사용
govc folder.create /Datacenter/vm/basphere-vms
```

#### 빠른 진단 스크립트

새 환경에서 빠르게 문제를 진단하는 스크립트:

```bash
#!/bin/bash
echo "=== Basphere 배포 진단 ==="

echo -n "1. CLI 설치: "
which basphere-admin > /dev/null && echo "OK" || echo "FAIL"

echo -n "2. API 바이너리: "
[ -f /opt/basphere/basphere-api/build/basphere-api-linux-amd64 ] && echo "OK" || echo "FAIL"

echo -n "3. config.yaml: "
[ -f /etc/basphere/config.yaml ] && echo "OK" || echo "FAIL"

echo -n "4. vsphere.env (권한): "
[ "$(stat -c %a /etc/basphere/vsphere.env 2>/dev/null)" = "600" ] && echo "OK" || echo "FAIL"

echo -n "5. api.yaml bastion 섹션: "
grep -q "^bastion:" /etc/basphere/api.yaml 2>/dev/null && echo "OK" || echo "FAIL"

echo -n "6. API 서버 상태: "
systemctl is-active basphere-api 2>/dev/null || echo "FAIL"

echo -n "7. SSH PasswordAuth: "
sudo sshd -T 2>/dev/null | grep -q "passwordauthentication no" && echo "OK (disabled)" || echo "WARNING (enabled)"

echo -n "8. fail2ban 상태: "
systemctl is-active fail2ban 2>/dev/null || echo "FAIL"

echo ""
echo "=== vSphere 연결 테스트 ==="
if [ -f /etc/basphere/vsphere.env ]; then
    source /etc/basphere/vsphere.env
    VCENTER=$(grep "server:" /etc/basphere/config.yaml | awk '{print $2}' | tr -d '"')
    echo "vCenter: $VCENTER"
    curl -k -s -o /dev/null -w "HTTP Status: %{http_code}\n" \
        -u "${VSPHERE_USER}:${VSPHERE_PASSWORD}" \
        "https://${VCENTER}/rest/com/vmware/cis/session" -X POST
fi
```
