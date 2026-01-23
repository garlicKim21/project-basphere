# Basphere 설치 가이드

다른 환경(회사 등)에 Basphere를 이식할 때 참고하세요.

## 1. Bastion 서버 준비

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

## 2. 환경별 설정 파일 수정

### /etc/basphere/config.yaml

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

### /etc/basphere/vsphere.env

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

## 3. 빌드 도구 설치

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

## 4. 설치 실행

```bash
# CLI 설치 (의존성 자동 설치: jq, yq, terraform)
cd /opt/basphere/basphere-cli
sudo ./install.sh

# API 서버 빌드 (/opt 디렉토리이므로 sudo 필요)
cd /opt/basphere/basphere-api
sudo make tidy && sudo make build-linux
```

## 5. VM 템플릿 준비

vCenter에 다음 조건을 만족하는 VM 템플릿이 필요합니다.

### Ubuntu 템플릿 (Cloud Image 사용)

```bash
# Ubuntu Cloud Image 다운로드 (OVA 형식)
# https://cloud-images.ubuntu.com/noble/current/
# noble-server-cloudimg-amd64.ova 다운로드 후 vSphere에 배포
```

### Rocky Linux 템플릿 (ISO 설치)

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

### 템플릿 요구사항

- cloud-init 설치 및 활성화
- open-vm-tools 설치
- cloud-utils-growpart 설치 (디스크 자동 확장)
- 네트워크 어댑터: VMXNET3
- 파티션: Standard (LVM 미사용 권장)

## 6. 환경별 체크리스트

| 항목 | 설정 예시 | 참고 |
|------|----------|------|
| Bastion IP | `<your-bastion-ip>` | 환경에 맞게 설정 |
| vCenter | `vcenter.example.com` | vCenter 주소 |
| Datacenter | `Your-Datacenter` | vSphere 데이터센터 이름 |
| 네트워크 대역 | `10.0.0.0/21` | 할당받은 대역 |
| MTU | 1500 (일반) / 1450 (오버레이) | 네트워크 환경에 따라 |
| DNS | `8.8.8.8` 또는 사내 DNS | 환경에 맞게 |

## 7. API 서버 설정

### /etc/basphere/api.yaml

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

### systemd 서비스 등록

```bash
sudo cp /opt/basphere/basphere-api/basphere-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable basphere-api
sudo systemctl start basphere-api
```

## 8. 외부 접근 설정 (선택사항)

Bastion 서버를 외부에서 접근 가능하게 설정할 때 참고하세요.

### 포트 포워딩 (방화벽/라우터)

```
외부 포트 50022 → 내부 <bastion-ip>:22
```

> **주의**: 비표준 포트(50022 등)를 사용하여 자동화된 공격을 줄일 수 있습니다.

### DNS 설정

| 레코드 | 값 | 참고 |
|--------|-----|------|
| `bastion.example.com` | A 레코드 → 공인 IP | Cloudflare 사용 시 **DNS Only** (회색 구름) |

> **Cloudflare 주의**: Proxied(주황색 구름) 모드는 HTTP/HTTPS만 지원하므로, SSH(TCP)는 **DNS Only**로 설정해야 합니다.

### api.yaml bastion 설정 예시

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

## 9. 설치 후 확인

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

## 10. 빠른 진단 스크립트

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

## 다음 단계

- 보안 설정: [security.md](security.md)
- 문제 해결: [troubleshooting.md](troubleshooting.md)
