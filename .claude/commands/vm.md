---
description: VM 생성/삭제/조회 (kimht 사용자로 실행)
allowed-tools: mcp__bastion__ssh_exec_sudo
argument-hint: <create|delete|list|show> [vm-name] [spec]
---

# VM 관리 (kimht 사용자)

kimht 사용자 권한으로 VM을 관리합니다.

## 사용법

- `/vm list` - VM 목록 조회
- `/vm create <name> [spec]` - VM 생성 (spec: small/medium/large, 기본: small)
- `/vm delete <name>` - VM 삭제
- `/vm show <name>` - VM 상세 정보

## 명령어 매핑

kimht 사용자로 전환하여 실행 (SUDO_USER 환경변수 초기화 필요):

### list
```bash
sudo -u kimht USER=kimht SUDO_USER= list-vms
```

### create
```bash
sudo -u kimht USER=kimht SUDO_USER= create-vm -n <name> -s <spec>
```

### delete
```bash
sudo -u kimht USER=kimht SUDO_USER= delete-vm <name>
```

### show
```bash
sudo -u kimht USER=kimht SUDO_USER= list-vms | grep <name>
```

## 인수 파싱

$ARGUMENTS 형식: `<action> [name] [spec]`

| 입력 | action | name | spec |
|------|--------|------|------|
| `list` | list | - | - |
| `create test1 medium` | create | test1 | medium |
| `create test1` | create | test1 | small |
| `delete test1` | delete | test1 | - |

## 결과 보고

- 생성 시: VM 이름, IP 주소, 스펙
- 삭제 시: 삭제 완료 확인
- 목록: 테이블 형식으로 표시
