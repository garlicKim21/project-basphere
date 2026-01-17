---
description: 테스트 VM 생성 또는 삭제 (Bastion CLI 사용)
allowed-tools: mcp__bastion__ssh_exec, mcp__bastion__ssh_exec_sudo
argument-hint: <create|delete|list> [vm-name] [spec]
---

# 테스트 VM 관리

Bastion의 CLI를 통해 테스트 VM을 생성/삭제/조회합니다.

## 사용법

- `/test-vm create myvm small` - small 스펙으로 myvm 생성
- `/test-vm delete myvm` - myvm 삭제
- `/test-vm list` - VM 목록 조회

## 명령어 매핑

### create
```bash
# basphere 사용자로 실행
sudo -u basphere create-vm -n <vm-name> -s <spec>
```

스펙 옵션: small, medium, large (기본: small)

### delete
```bash
sudo -u basphere delete-vm <vm-name>
```

### list
```bash
sudo -u basphere list-vms
```

## 인수 파싱

$ARGUMENTS 형식: `<action> [name] [spec]`

예시:
- `create test-vm1 medium` → action=create, name=test-vm1, spec=medium
- `delete test-vm1` → action=delete, name=test-vm1
- `list` → action=list

## 결과 보고

- VM 생성: IP 주소, 스펙 정보
- VM 삭제: 삭제 확인
- VM 목록: 테이블 형식으로 표시
