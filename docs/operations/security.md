# Basphere 보안 설정

Bastion 서버의 보안을 강화하기 위한 설정 가이드입니다.

## SSH 보안 설정

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

> **파일명 주의**: 파일명이 `40-`으로 시작해야 cloud-init의 `50-cloud-init.conf`보다 먼저 로드되어 설정이 적용됩니다.

## fail2ban 설치 및 설정

### 설치

```bash
sudo apt install -y fail2ban
```

### SSH jail 설정

```bash
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
```

### 서비스 시작

```bash
sudo systemctl restart fail2ban
sudo systemctl enable fail2ban

# 상태 확인
sudo fail2ban-client status sshd
```

## 보안 설정 요약

| 설정 | 값 | 설명 |
|------|-----|------|
| `PasswordAuthentication` | no | SSH 키 인증만 허용 |
| `PermitRootLogin` | no | root 로그인 차단 |
| `MaxAuthTries` | 3 | 인증 시도 제한 |
| `MaxSessions` | 20 | 다중 세션 허용 |
| `MaxStartups` | 10:50:30 | DoS 방지 |
| fail2ban `maxretry` | 3 | 3회 실패 시 차단 |
| fail2ban `bantime` | 30분 | 차단 시간 |

## vSphere 인증 정보 보호

vSphere 인증 정보는 반드시 root만 읽을 수 있어야 합니다:

```bash
sudo chmod 600 /etc/basphere/vsphere.env
sudo chown root:root /etc/basphere/vsphere.env
```

**아키텍처**:
```
┌─────────────┐     HTTP      ┌─────────────┐    Terraform    ┌─────────────┐
│    CLI      │──────────────▶│  API Server │────────────────▶│   vSphere   │
│  (사용자)   │               │   (root)    │                 │  (vCenter)  │
└─────────────┘               └──────┬──────┘                 └─────────────┘
                                     │
                                     ▼
                              vsphere.env
                              (600, root만 읽기)
```

## nginx 보안 설정

nginx 리버스 프록시를 통해 내부 API를 보호합니다:

| 엔드포인트 | 외부 접근 | 설명 |
|-----------|----------|------|
| `/register` | ✅ | 사용자 등록 페이지 |
| `/success` | ✅ | 등록 완료 페이지 |
| `/health` | ✅ | 헬스체크 |
| `/api/v1/vms` | ❌ | VM 관리 API (내부 전용) |
| `/api/v1/pending` | ❌ | 대기 목록 (내부 전용) |

자세한 nginx 설정은 [deploy/README.md](../../deploy/README.md)를 참조하세요.

## 설정 확인

```bash
# SSH 보안 설정 확인
sudo sshd -T | grep -E "^(passwordauthentication|permitrootlogin|maxsessions)"

# fail2ban 상태 확인
sudo fail2ban-client status sshd

# vsphere.env 권한 확인
ls -la /etc/basphere/vsphere.env
```

## 관련 문서

- [설치 가이드](installation.md)
- [트러블슈팅](troubleshooting.md)
