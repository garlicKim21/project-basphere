---
description: Basphere 관리자 명령어 실행 (basphere-admin)
allowed-tools: mcp__bastion__ssh_exec_sudo
argument-hint: <user list|user show|user pending|user approve|user reject> [username]
---

# Basphere 관리자 명령어

basphere 계정으로 관리자 CLI를 실행합니다.

## 사용법

- `/admin user list` - 사용자 목록
- `/admin user show <username>` - 사용자 상세 정보
- `/admin user pending` - 대기 중인 등록 요청
- `/admin user approve <username>` - 등록 승인
- `/admin user reject <username>` - 등록 거부

## 명령어 매핑

$ARGUMENTS를 그대로 `basphere-admin`에 전달:

```bash
sudo basphere-admin $ARGUMENTS
```

## 예시

| 입력 | 실행 명령 |
|------|-----------|
| `/admin user list` | `basphere-admin user list` |
| `/admin user show kimht` | `basphere-admin user show kimht` |
| `/admin user pending` | `basphere-admin user pending` |

## 결과 보고

- 명령 실행 결과를 그대로 표시
- 에러 발생 시 상세 메시지 포함
