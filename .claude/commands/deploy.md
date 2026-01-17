---
description: Bastion 서버에 최신 코드 배포 (git pull + CLI 재설치)
allowed-tools: Bash, mcp__bastion__ssh_exec, mcp__bastion__ssh_exec_sudo
---

# Bastion 배포

Bastion 서버(/opt/basphere)에 최신 코드를 배포합니다.

## 단계

1. **로컬 변경사항 확인**: 커밋되지 않은 변경이 있으면 먼저 커밋/푸시 권유
2. **Bastion에서 git pull**: MCP를 통해 `/opt/basphere`에서 `git pull` 실행
3. **CLI 재설치**: `./install.sh` 실행하여 CLI 업데이트
4. **결과 확인**: 설치 성공 여부 보고

## 실행 명령

```bash
# Bastion에서 실행할 명령들
cd /opt/basphere && sudo git pull
cd /opt/basphere/basphere-cli && sudo ./install.sh
```

## 주의사항

- sudo 권한 필요
- 네트워크 연결 확인 필요
- 실패 시 에러 메시지 상세히 보고

$ARGUMENTS가 "api"를 포함하면 API 서버도 빌드:
```bash
cd /opt/basphere/basphere-api && make tidy && make build-linux
```
