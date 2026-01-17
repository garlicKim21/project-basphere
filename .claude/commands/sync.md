---
description: 로컬 변경사항을 Bastion에 동기화 (commit + push + deploy)
allowed-tools: Bash, mcp__bastion__ssh_exec, mcp__bastion__ssh_exec_sudo
argument-hint: [commit-message]
---

# 로컬 → Bastion 동기화

로컬 변경사항을 커밋하고 Bastion에 자동 배포합니다.

## 단계

1. **변경사항 확인**: `git status`로 변경 파일 확인
2. **커밋**: $ARGUMENTS를 커밋 메시지로 사용 (없으면 자동 생성)
3. **푸시**: origin에 푸시
4. **Bastion 배포**: MCP로 git pull + install.sh 실행

## 커밋 메시지

- $ARGUMENTS가 제공되면 그대로 사용
- 없으면 변경 내용 분석하여 Conventional Commits 형식으로 자동 생성

## 실행 흐름

```bash
# 로컬
git add -A
git commit -m "커밋 메시지"
git push

# Bastion (MCP)
cd /opt/basphere && sudo git pull
cd /opt/basphere/basphere-cli && sudo ./install.sh
```

## 결과 보고

- 커밋 해시
- 변경된 파일 수
- Bastion 배포 성공 여부
