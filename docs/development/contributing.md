# Basphere 개발 가이드

## 개발 규칙

### Bash 스크립트

- `set -euo pipefail` 필수
- 함수명: snake_case (예: `create_vm`, `get_config`)
- 로그: `log_info`, `log_success`, `log_warn`, `log_error` 사용
- ShellCheck 경고 없어야 함

```bash
#!/bin/bash
set -euo pipefail

source /usr/local/lib/basphere/common.sh

function create_vm() {
    log_info "VM 생성 중: $1"
    # ...
    log_success "VM 생성 완료"
}
```

### Go 코드

- `gofmt` 적용
- 에러는 반드시 처리
- 인터페이스로 추상화 (Store, Provisioner)

```go
// 인터페이스 정의
type Store interface {
    Create(req *RegistrationRequest) error
    Get(id string) (*RegistrationRequest, error)
    // ...
}

// 구현
var _ Store = (*FileStore)(nil)
var _ Store = (*PostgresStore)(nil)  // IDP 이후
```

### 커밋 메시지

Conventional Commits 형식:

```
feat: Add MTU configuration support
fix: Use write_files for cloud-init MTU config
docs: Update installation guide
refactor: Extract IPAM logic to separate module
```

| 타입 | 설명 |
|------|------|
| `feat` | 새로운 기능 |
| `fix` | 버그 수정 |
| `docs` | 문서 변경 |
| `refactor` | 코드 리팩토링 |
| `test` | 테스트 추가/수정 |
| `chore` | 빌드, 설정 등 |

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

## 디렉토리 구조

```
project-basphere/
├── basphere-cli/           # Bash 기반 CLI 도구
│   ├── scripts/
│   │   ├── basphere-admin  # 관리자 CLI
│   │   ├── user/           # 사용자 CLI
│   │   └── internal/       # 내부 스크립트 (IPAM)
│   ├── lib/common.sh       # 공통 함수
│   └── templates/terraform/ # Terraform 템플릿
│
├── basphere-api/           # Go REST API 서버
│   ├── cmd/basphere-api/   # 서버 진입점
│   ├── internal/
│   │   ├── handler/        # HTTP 핸들러
│   │   ├── model/          # 데이터 모델
│   │   ├── store/          # 저장소
│   │   └── provisioner/    # bash 스크립트 호출
│   └── web/templates/      # HTML 템플릿
│
├── docs/                   # 문서
│   ├── design/             # IDP 설계 문서
│   ├── operations/         # 운영 가이드
│   └── development/        # 개발 가이드
│
└── deploy/                 # 배포 설정
    ├── nginx/              # nginx 설정
    └── systemd/            # systemd 서비스
```

## 주의사항

- vSphere customization과 cloud-init을 함께 사용 시 네트워크 설정 충돌 주의
- snap으로 설치된 yq는 /etc 접근 불가 (바이너리 버전 사용)
- Terraform 상태 파일은 로컬 저장 (각 사용자별 디렉토리)
- **Ubuntu 24.04 cloud-init 네트워크 설정**: 네트워크 설정은 `guestinfo.metadata` 안에 `network` 키로 포함해야 함. 별도의 `guestinfo.network`는 작동하지 않음

## 테스트

### CLI 테스트

```bash
# 사용자로 전환하여 테스트
su - testuser

# VM 생성 테스트
create-vm -n test-vm -s tiny -o ubuntu-24.04

# VM 조회
list-vms

# VM 삭제
delete-vm test-vm
```

### API 테스트

```bash
# 개발 모드 실행 (Mock provisioner)
cd basphere-api
make dev

# 헬스 체크
curl http://localhost:8080/health

# 등록 API 테스트
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "team": "DevOps",
    "public_key": "ssh-ed25519 AAAA..."
  }'
```

## 관련 문서

- [설치 가이드](../operations/installation.md)
- [트러블슈팅](../operations/troubleshooting.md)
- [CLI README](../../basphere-cli/README.md)
- [API README](../../basphere-api/README.md)
