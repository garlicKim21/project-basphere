# SSH 키 가이드 (macOS)

macOS에서 SSH 키를 생성하고 관리하는 방법입니다.

---

## 목차

1. [처음 SSH 키를 만드는 경우](#1-처음-ssh-키를-만드는-경우)
2. [이미 SSH 키가 있는 경우](#2-이미-ssh-키가-있는-경우)
3. [새 키를 별도로 만들어 사용하는 경우](#3-새-키를-별도로-만들어-사용하는-경우)
4. [SSH Config 설정으로 편리하게 접속하기](#4-ssh-config-설정으로-편리하게-접속하기)
5. [트러블슈팅](#5-트러블슈팅)

---

## 1. 처음 SSH 키를 만드는 경우

SSH 키가 전혀 없는 경우, 새로 생성합니다.

### 1.1 터미널 열기

- `Command + Space`를 눌러 Spotlight 실행
- "터미널" 또는 "Terminal" 입력 후 엔터

### 1.2 SSH 키 생성

```bash
ssh-keygen -t ed25519 -C "your-email@example.com"
```

실행 결과:
```
Generating public/private ed25519 key pair.
Enter file in which to save the key (/Users/username/.ssh/id_ed25519): [엔터]
Enter passphrase (empty for no passphrase): [비밀번호 입력 또는 엔터]
Enter same passphrase again: [비밀번호 재입력 또는 엔터]
```

> **팁**: 비밀번호(passphrase)는 선택사항입니다. 보안을 위해 설정을 권장합니다.

### 1.3 공개키 확인 및 복사

```bash
cat ~/.ssh/id_ed25519.pub
```

출력 예시:
```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG... your-email@example.com
```

이 전체 내용을 복사하여 Basphere 등록 폼의 "SSH 공개키" 필드에 붙여넣습니다.

### 1.4 클립보드에 복사 (선택)

```bash
cat ~/.ssh/id_ed25519.pub | pbcopy
```

> `pbcopy`는 출력 내용을 클립보드에 복사합니다. 이후 `Command + V`로 붙여넣기 가능.

---

## 2. 이미 SSH 키가 있는 경우

기존에 생성한 SSH 키가 있다면 그대로 사용할 수 있습니다.

### 2.1 기존 키 확인

```bash
ls -la ~/.ssh/
```

출력 예시:
```
-rw-------  1 username  staff   411 Jan 15 10:00 id_ed25519
-rw-r--r--  1 username  staff   100 Jan 15 10:00 id_ed25519.pub
```

다음 파일 중 하나가 있으면 키가 존재합니다:
- `id_ed25519` / `id_ed25519.pub` (Ed25519 - 권장)
- `id_rsa` / `id_rsa.pub` (RSA)

### 2.2 공개키 확인 및 복사

```bash
# Ed25519 키인 경우
cat ~/.ssh/id_ed25519.pub

# RSA 키인 경우
cat ~/.ssh/id_rsa.pub
```

출력된 내용 전체를 Basphere 등록 폼에 붙여넣습니다.

---

## 3. 새 키를 별도로 만들어 사용하는 경우

기존 키는 유지하면서 Basphere 전용 키를 별도로 만들고 싶은 경우입니다.

### 3.1 새 키 생성 (다른 파일명)

```bash
ssh-keygen -t ed25519 -C "basphere" -f ~/.ssh/id_basphere
```

결과:
- 개인키: `~/.ssh/id_basphere`
- 공개키: `~/.ssh/id_basphere.pub`

### 3.2 공개키 확인

```bash
cat ~/.ssh/id_basphere.pub
```

이 내용을 Basphere 등록 폼에 붙여넣습니다.

### 3.3 접속 시 키 지정 (-i 옵션)

```bash
ssh -i ~/.ssh/id_basphere username@bastion-server
```

예시:
```bash
ssh -i ~/.ssh/id_basphere kimht@bastion.company.local
```

> **불편함**: 매번 `-i` 옵션을 입력해야 합니다. [SSH Config 설정](#4-ssh-config-설정으로-편리하게-접속하기)을 사용하면 편리합니다.

---

## 4. SSH Config 설정으로 편리하게 접속하기

SSH config 파일을 설정하면 간단한 명령으로 접속할 수 있습니다.

### 4.1 config 파일 생성/편집

```bash
nano ~/.ssh/config
```

또는 선호하는 에디터 사용:
```bash
code ~/.ssh/config   # VS Code
vim ~/.ssh/config    # Vim
```

### 4.2 설정 내용 추가

```
# Basphere Bastion 서버
Host bastion
    HostName bastion.company.local
    User kimht
    IdentityFile ~/.ssh/id_basphere

# Basphere VM 접속 (Bastion 경유)
Host 10.254.0.*
    ProxyJump bastion
    User ubuntu
    IdentityFile ~/.ssh/id_basphere
```

> **참고**: `IdentityFile`은 기본 키(`~/.ssh/id_ed25519`)를 사용하는 경우 생략 가능합니다.

### 4.3 config 파일 권한 설정

```bash
chmod 600 ~/.ssh/config
```

### 4.4 간편 접속

설정 후 다음과 같이 간단히 접속:

```bash
# Bastion 접속
ssh bastion

# VM 직접 접속 (Bastion 경유 자동)
ssh 10.254.0.32
```

### 4.5 Rocky Linux VM 접속 설정

Rocky Linux VM의 기본 사용자는 `rocky`입니다. 별도 설정 추가:

```
# Rocky Linux VM
Host rocky-*
    ProxyJump bastion
    User rocky
    IdentityFile ~/.ssh/id_basphere
```

---

## 5. 트러블슈팅

### 권한 오류 (Permission denied)

```
Permission denied (publickey).
```

**원인**: 공개키가 서버에 등록되지 않았거나 키 파일 권한 문제

**해결**:
1. 등록 시 올바른 공개키를 제출했는지 확인
2. 키 파일 권한 확인:
   ```bash
   chmod 600 ~/.ssh/id_ed25519
   chmod 644 ~/.ssh/id_ed25519.pub
   ```

### 키를 찾을 수 없음

```
Warning: Identity file ~/.ssh/id_basphere not found.
```

**해결**: 파일 경로 확인
```bash
ls -la ~/.ssh/
```

### 연결 시간 초과 (Connection timed out)

```
ssh: connect to host bastion.company.local port 22: Connection timed out
```

**원인**: 네트워크 문제 또는 서버 주소 오류

**해결**:
1. 서버 주소 확인
2. VPN 연결 확인 (필요한 경우)
3. 방화벽 설정 확인

### SSH Agent 사용 (passphrase 자동 입력)

키에 비밀번호를 설정한 경우, SSH Agent를 사용하면 편리합니다:

```bash
# SSH Agent 시작 (macOS는 기본 실행됨)
eval "$(ssh-agent -s)"

# 키 추가
ssh-add ~/.ssh/id_basphere

# 키 목록 확인
ssh-add -l
```

> macOS Keychain을 사용하면 재부팅 후에도 비밀번호가 저장됩니다:
> ```bash
> ssh-add --apple-use-keychain ~/.ssh/id_basphere
> ```

---

## 다음 단계

SSH 키 준비가 완료되면 [사용자 가이드](user-guide.md)로 돌아가 계정 등록을 진행하세요.
