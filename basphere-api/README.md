# Basphere API Server

사용자 등록 요청을 위한 웹 API 서버입니다.

## 개요

- 사용자가 웹 폼을 통해 등록 요청을 제출
- 관리자가 CLI를 통해 요청을 승인/거부
- 승인 시 자동으로 계정 생성 (basphere-admin 호출)

## 구조

```
basphere-api/
├── cmd/basphere-api/
│   └── main.go              # 서버 진입점
├── internal/
│   ├── config/              # 설정 로딩
│   ├── handler/             # HTTP 핸들러
│   ├── model/               # 데이터 모델
│   ├── store/               # 저장소 인터페이스
│   └── provisioner/         # 사용자 프로비저닝
├── web/templates/           # HTML 템플릿
├── config/                  # 설정 파일 예시
├── Makefile
└── basphere-api.service     # systemd 서비스
```

## 빌드

```bash
# 의존성 설치
make tidy

# 빌드
make build

# Linux용 크로스 컴파일
make build-linux
```

## 실행

### 개발 모드

```bash
# Mock provisioner 사용 (실제 계정 생성 안 함)
make dev
```

### 프로덕션 모드

```bash
# 직접 실행
./build/basphere-api --config /etc/basphere/api.yaml

# 또는 systemd 사용
sudo cp basphere-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable basphere-api
sudo systemctl start basphere-api
```

## API 엔드포인트

### 웹 페이지

| 경로 | 설명 |
|------|------|
| GET `/` | 등록 페이지로 리다이렉트 |
| GET `/register` | 등록 폼 |
| POST `/register` | 등록 폼 제출 |
| GET `/success` | 등록 성공 페이지 |

### REST API

| Method | 경로 | 설명 |
|--------|------|------|
| POST | `/api/v1/register` | 등록 요청 (JSON) |
| GET | `/api/v1/pending` | 대기 중인 요청 목록 |
| GET | `/api/v1/pending/{username}` | 특정 요청 조회 |
| POST | `/api/v1/users/{username}/approve` | 요청 승인 |
| POST | `/api/v1/users/{username}/reject` | 요청 거부 |
| GET | `/health` | 헬스 체크 |

### API 예시

```bash
# 등록 요청 (API)
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "hong",
    "email": "hong@company.com",
    "team": "DevOps",
    "public_key": "ssh-ed25519 AAAA... hong@macbook"
  }'

# 대기 목록 조회
curl http://localhost:8080/api/v1/pending

# 승인
curl -X POST http://localhost:8080/api/v1/users/hong/approve

# 거부
curl -X POST http://localhost:8080/api/v1/users/hong/reject \
  -H "Content-Type: application/json" \
  -d '{"reason": "중복 요청"}'
```

## 설정

`/etc/basphere/api.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

storage:
  pending_dir: "/var/lib/basphere/pending"

provisioner:
  admin_script: "/usr/local/bin/basphere-admin"
```

## CLI와 연동

관리자는 웹 API 대신 CLI로도 요청을 관리할 수 있습니다:

```bash
# 대기 중인 요청 목록
sudo basphere-admin user pending

# 특정 요청 상세 조회
sudo basphere-admin user pending hong

# 승인
sudo basphere-admin user approve hong

# 거부
sudo basphere-admin user reject hong --reason "중복 요청"
```

## IDP 마이그레이션

이 API 서버는 향후 IDP 구축 시 다음과 같이 재사용됩니다:

1. **API 엔드포인트**: 동일한 REST API 구조 유지
2. **Store 인터페이스**: `FileStore` → `PostgresStore`로 교체
3. **Provisioner**: bash 스크립트 호출 → Go 네이티브 구현

```go
// 현재
type FileStore struct { ... }

// IDP
type PostgresStore struct { ... }

// 동일한 인터페이스 구현
var _ store.Store = (*FileStore)(nil)
var _ store.Store = (*PostgresStore)(nil)
```
