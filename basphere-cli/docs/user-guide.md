# Basphere 사용자 가이드

개발자를 위한 VM 셀프서비스 사용 가이드

---

## 시작하기

### 1. SSH 키 준비

Basphere는 SSH 키 인증을 사용합니다. 등록 전에 SSH 키가 필요합니다.

- [macOS SSH 키 가이드](ssh-guide-macos.md)
- [Windows SSH 키 가이드](ssh-guide-windows.md)

### 2. 계정 등록 요청

웹 브라우저에서 등록 페이지에 접속하여 계정을 신청합니다.

```
http://<bastion-server>:8080/register
```

등록 폼에 입력할 정보:
- **사용자 ID**: 원하는 계정명 (영문 소문자, 숫자, 하이픈)
- **이메일**: 연락 가능한 이메일 주소
- **소속 팀**: 소속 팀/부서명
- **SSH 공개키**: 준비한 SSH 공개키 내용 (예: `ssh-ed25519 AAAA...`)

> **공개키 확인 방법**: 터미널에서 `cat ~/.ssh/id_ed25519.pub` 실행

### 3. 승인 대기

등록 요청이 제출되면 관리자의 승인을 기다립니다. 승인이 완료되면 Bastion 서버에 접속할 수 있습니다.

### 4. Bastion 서버 접속

승인 후 SSH로 Bastion 서버에 접속합니다.

```bash
ssh <user-id>@<bastion-server>
```

예시:
```bash
ssh <user-id>@bastion.example.com
```

> **접속 오류 시**: [SSH 키 가이드](ssh-guide-macos.md)의 트러블슈팅 섹션을 참조하세요.

### 5. 사용 가능한 명령어

| 명령어 | 설명 |
|--------|------|
| `create-vm` | VM 생성 |
| `list-vms` | VM 목록 조회 |
| `delete-vm <name>` | VM 삭제 |
| `list-resources` | 전체 리소스 조회 |
| `show-quota` | 할당량 확인 |

---

## VM 생성

### 대화형 모드 (권장)

```bash
create-vm
```

실행 예시:
```
=== VM 생성 ===

? VM 이름: my-dev-server

? OS 선택
  [1] ubuntu-24.04 - Ubuntu 24.04 LTS (Noble)
  [2] rocky-10.1 - Rocky Linux 10.1
? 선택 [1-2]: 1

? 스펙 선택
  [1] tiny - 2 vCPU, 4GB RAM, 50GB Disk
  [2] small - 2 vCPU, 8GB RAM, 50GB Disk
  [3] medium - 4 vCPU, 16GB RAM, 100GB Disk
  [4] large - 8 vCPU, 32GB RAM, 200GB Disk
  [5] huge - 16 vCPU, 64GB RAM, 200GB Disk
? 선택 [1-5]: 2

? 생성할 VM 대수 [1]: 1

생성할 VM 정보:
  - 이름: my-dev-server
  - OS: ubuntu-24.04
  - 스펙: small (2 vCPU, 8GB RAM, 50GB Disk)
  - 대수: 1

? VM을 생성하시겠습니까? [Y/n]: Y

[INFO] VM 생성 준비 중: my-dev-server
[OK] IP 할당: 10.254.0.32
[INFO] VM 생성 중... (몇 분 소요될 수 있습니다)
[OK] VM 생성 완료: my-dev-server (10.254.0.32)
```

### 명령행 모드

미리 옵션을 지정하여 빠르게 생성:

```bash
# 단일 VM 생성 (Ubuntu 기본)
create-vm -n my-server -s small

# OS 지정하여 생성
create-vm -n rocky-server -s small -o rocky-10.1

# 여러 VM 생성
create-vm -n web-server -s medium -c 3
```

옵션:
- `-n, --name <name>`: VM 이름
- `-s, --spec <spec>`: 스펙 (tiny, small, medium, large, huge)
- `-o, --os <os>`: OS 선택 (ubuntu-24.04, rocky-10.1) - 기본값: ubuntu-24.04
- `-c, --count <count>`: 생성할 VM 수 (기본값: 1)

> **VM 이름 규칙**
> - 영문 소문자로 시작
> - 영문 소문자, 숫자, 하이픈(-) 사용 가능
> - 연속된 하이픈(--) 불가
> - 최대 63자
>
> **참고**: vSphere에서는 사용자 이름이 자동으로 VM 이름 앞에 붙습니다.
> 예: `my-server` → vSphere에서 `<user-id>-my-server`로 표시

### 여러 VM 생성 시

`-c` 옵션으로 여러 대를 생성하면 자동으로 번호가 붙습니다:

```bash
create-vm -n web -s small -c 3
```

결과:
- CLI: `web-1`, `web-2`, `web-3`
- vSphere: `<user-id>-web-1`, `<user-id>-web-2`, `<user-id>-web-3`
- IP: 10.254.0.32, 10.254.0.33, 10.254.0.34

---

## VM 조회

### VM 목록

```bash
list-vms
```

출력 예시:
```
=== VM 목록 (<user-id>) ===

NAME                 IP               SPEC       STATUS
--------------------------------------------------------------------------------
my-dev-server        10.254.0.32      small      running
web-1                10.254.0.33      medium     running
web-2                10.254.0.34      medium     running

총 3개 VM
할당량: 3 / 10
```

### 상세 정보 포함

```bash
list-vms -a
```

생성 날짜 등 추가 정보를 표시합니다.

### JSON 형식

```bash
list-vms -j
```

스크립트에서 사용하기 좋은 JSON 형식으로 출력합니다.

---

## VM 삭제

```bash
delete-vm <vm-name>
```

예시:
```bash
delete-vm my-dev-server
```

출력:
```
삭제할 VM 정보:
  - 이름: my-dev-server
  - IP: 10.254.0.32
  - 스펙: small
  - 상태: running

? 정말로 VM 'my-dev-server'을(를) 삭제하시겠습니까? [y/N]: y

[INFO] VM 삭제 중: my-dev-server
[INFO] Terraform destroy 실행 중...
[OK] IP 반환 완료: 10.254.0.32
[OK] VM 삭제 완료: my-dev-server
```

> **주의**: 삭제된 VM은 복구할 수 없습니다. 중요한 데이터는 미리 백업하세요.

---

## 리소스 조회

### 전체 리소스

VM과 Kubernetes 클러스터를 함께 조회:

```bash
list-resources
```

출력 예시:
```
=== 리소스 목록 (<user-id>) ===

--- VMs ---
NAME                 IP               SPEC       STATUS
--------------------------------------------------------------------------------
my-dev-server        10.254.0.32      small      running
web-1                10.254.0.33      medium     running

--- Kubernetes Clusters ---
생성된 클러스터가 없습니다.
(클러스터 생성은 Stage 2에서 지원됩니다)

==========================================
총 리소스: VM 2개, 클러스터 0개
```

---

## 할당량 확인

```bash
show-quota
```

출력 예시:
```
=== 할당량 (<user-id>) ===

IP 블록: 10.254.0.32 (32개)

리소스 사용량:

  VMs:       [██████░░░░░░░░░░░░░░░░░░░░░░░░] 2/10 (20%)
  Clusters:  [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░] 0/3 (0%)
  IPs:       [█░░░░░░░░░░░░░░░░░░░░░░░░░░░░░] 2/32 (6%)
```

### JSON 형식

```bash
show-quota -j
```

---

## VM 접속

VM이 생성되면 SSH로 접속할 수 있습니다.

```bash
ssh <user-id>@<vm-ip>
```

예시:
```bash
ssh <user-id>@10.254.0.32
```

> **사용자 계정**: Bastion과 동일한 사용자 ID 사용
> **인증 방식**: Bastion에 등록된 SSH 키와 동일한 키 사용

### Bastion을 통한 접속 (ProxyJump)

로컬에서 직접 VM에 접속하려면 `~/.ssh/config`에 다음과 같이 설정합니다:

```
Host bastion
    HostName <bastion-address>
    User <user-id>
    Port <bastion-port>
    ServerAliveInterval 30
    ServerAliveCountMax 3
    TCPKeepAlive yes

Host 10.254.0.*
    ProxyJump bastion
    User <user-id>
```

> **설정값 안내**: `<bastion-address>`, `<bastion-port>`는 관리자에게 문의하거나 등록 완료 페이지에서 확인하세요.

설정 후 VM에 바로 접속:
```bash
ssh 10.254.0.32
```

또는 명령어로 직접 접속:
```bash
# 포트가 22인 경우
ssh -J <user-id>@<bastion-address> <user-id>@<vm-ip>

# 포트가 22가 아닌 경우 (예: 50022)
ssh -J <user-id>@<bastion-address>:<port> <user-id>@<vm-ip>
```

---

## VM 스펙

| 스펙 | vCPU | 메모리 | 디스크 | 용도 |
|------|------|--------|--------|------|
| tiny | 2 | 4GB | 50GB | 테스트용 |
| small | 2 | 8GB | 50GB | 개발용 |
| medium | 4 | 16GB | 100GB | 일반 워크로드 |
| large | 8 | 32GB | 200GB | 고성능 워크로드 |
| huge | 16 | 64GB | 200GB | 대규모 워크로드 |

## 지원 OS

| OS | 옵션값 |
|----|--------|
| Ubuntu 24.04 LTS | `ubuntu-24.04` |
| Rocky Linux 10.1 | `rocky-10.1` |

> **참고**: VM 접속 시 Bastion과 동일한 사용자 ID를 사용합니다.

---

## 제한사항

### 할당량

| 리소스 | 기본 제한 |
|--------|----------|
| VM | 최대 10개 |
| Kubernetes 클러스터 | 최대 3개 |
| IP 주소 | 최대 32개 |

할당량 증가가 필요하면 관리자에게 문의하세요.

### VM 이름

- 사용자별로 고유해야 함
- 영문 소문자, 숫자, 하이픈만 사용 가능
- 최대 63자

### 네트워크

- 각 사용자에게 독립된 IP 블록이 할당됨
- VM은 할당된 블록 내의 IP를 자동으로 받음

### vSphere 구조

- 각 사용자의 VM은 vSphere에서 별도 폴더에 저장됨 (`basphere-vms/<username>/`)
- VM 이름에는 사용자 이름이 자동으로 붙음 (CLI에서는 짧은 이름 사용)
- 이를 통해 vCenter에서 사용자별 리소스 구분 가능

---

## 자주 묻는 질문

### Q: VM 생성이 실패했어요

**A**: 몇 가지 원인이 있을 수 있습니다:

1. **할당량 초과**: `show-quota`로 확인
2. **이름 중복**: 동일한 이름의 VM이 이미 존재
3. **인프라 문제**: 관리자에게 문의

실패한 VM은 `list-vms`에서 `failed` 상태로 표시됩니다. 삭제 후 다시 시도하세요:
```bash
delete-vm <failed-vm-name>
create-vm
```

### Q: VM에 SSH 접속이 안 돼요

**A**: 다음을 확인하세요:

1. VM 상태가 `running`인지 확인 (`list-vms`)
2. VM 부팅 완료 대기 (생성 후 1-2분)
3. 올바른 사용자명 사용 (Bastion과 동일한 사용자 ID)
4. SSH 키가 올바른지 확인

### Q: VM의 IP 주소를 변경할 수 있나요?

**A**: 현재는 지원하지 않습니다. VM을 삭제하고 다시 생성하면 새 IP가 할당됩니다.

### Q: VM 스펙을 변경할 수 있나요?

**A**: 현재는 지원하지 않습니다. 다른 스펙의 VM을 새로 생성해야 합니다.

### Q: 데이터 백업은 어떻게 하나요?

**A**: VM 내부에서 직접 백업하세요:
```bash
# 예: 중요 파일을 다른 서버로 복사
scp -r /important/data user@backup-server:/backup/
```

---

## 도움말

각 명령어의 도움말:

```bash
create-vm --help
list-vms --help
delete-vm --help
show-quota --help
list-resources --help
```

문제가 있으면 관리자에게 문의하세요.
