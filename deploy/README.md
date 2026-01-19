# Basphere 배포 설정

이 디렉토리에는 Basphere를 프로덕션 환경에 배포하기 위한 설정 파일들이 포함되어 있습니다.

## 디렉토리 구조

```
deploy/
├── README.md           # 이 문서
├── nginx/
│   └── basphere.conf   # nginx 리버스 프록시 설정
└── systemd/
    └── basphere-api.service  # API 서버 systemd 서비스
```

## 아키텍처

```
인터넷
    │
    ▼ (80/443)
┌─────────────────────────────────────────┐
│            Bastion 서버                  │
│                                         │
│   ┌─────────────┐    ┌───────────────┐  │
│   │   nginx     │───▶│  basphere-api │  │
│   │ (0.0.0.0:80)│    │(127.0.0.1:8080)│  │
│   └─────────────┘    └───────────────┘  │
│                                         │
│   공개 엔드포인트:     내부 전용:         │
│   - /register         - /api/v1/vms     │
│   - /success          - /api/v1/quota   │
│   - /ssh-guide        - /api/v1/pending │
│   - /api/v1/register  - /api/v1/users/* │
│   - /health                             │
└─────────────────────────────────────────┘
```

## 설치 순서

### 1. API 서버 설정

```bash
# API 서버 빌드
cd /opt/basphere/basphere-api
make tidy && make build-linux

# api.yaml에서 바인딩 주소 확인 (127.0.0.1 권장)
sudo vim /etc/basphere/api.yaml
# server:
#   host: "127.0.0.1"  # localhost만 수신
#   port: 8080

# systemd 서비스 설치
sudo cp /opt/basphere/deploy/systemd/basphere-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable basphere-api
sudo systemctl start basphere-api

# 상태 확인
sudo systemctl status basphere-api
curl http://127.0.0.1:8080/health
```

### 2. nginx 설치 및 설정

```bash
# nginx 설치
sudo apt update
sudo apt install -y nginx

# 기본 설정 비활성화
sudo rm -f /etc/nginx/sites-enabled/default

# Rate limiting zone 추가 (http 블록 안에)
sudo vim /etc/nginx/nginx.conf
# http 블록 안에 다음 추가:
#   limit_req_zone $binary_remote_addr zone=basphere_general:10m rate=10r/s;
#   limit_req_zone $binary_remote_addr zone=basphere_register:10m rate=1r/s;

# Basphere 설정 복사
sudo cp /opt/basphere/deploy/nginx/basphere.conf /etc/nginx/sites-available/basphere
sudo ln -s /etc/nginx/sites-available/basphere /etc/nginx/sites-enabled/

# 설정 검증
sudo nginx -t

# nginx 재시작
sudo systemctl reload nginx
sudo systemctl enable nginx
```

### 3. 방화벽 설정 (선택사항)

```bash
# UFW 사용 시
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS (TLS 사용 시)
sudo ufw enable

# 8080 포트는 열지 않음 (nginx가 내부적으로 접근)
```

### 4. TLS/HTTPS 설정 (권장)

```bash
# Let's Encrypt certbot 설치
sudo apt install -y certbot python3-certbot-nginx

# 인증서 발급 (도메인 필요)
sudo certbot --nginx -d your-domain.com

# 자동 갱신 테스트
sudo certbot renew --dry-run
```

## 설정 파일 설명

### nginx/basphere.conf

nginx 리버스 프록시 설정:
- 공개 엔드포인트만 외부에 노출
- 내부 API (VM 관리, 할당량 등)는 차단
- 보안 헤더 추가
- Rate limiting 적용:
  - 일반 요청: 10r/s (초당 10개 요청)
  - 등록 API: 1r/s (초당 1개 요청, 봇 방지)

### systemd/basphere-api.service

API 서버 systemd 서비스:
- root 권한으로 실행 (Terraform, vsphere.env 접근)
- 실패 시 자동 재시작
- journald 로깅

## 엔드포인트 접근 권한

| 엔드포인트 | 외부 (nginx) | 내부 (localhost) | 설명 |
|-----------|-------------|-----------------|------|
| `/register` | ✅ | ✅ | 사용자 등록 페이지 |
| `/success` | ✅ | ✅ | 등록 완료 페이지 |
| `/ssh-guide` | ✅ | ✅ | SSH 키 가이드 |
| `/health` | ✅ | ✅ | 헬스체크 |
| `POST /api/v1/register` | ✅ | ✅ | 등록 API |
| `/api/v1/vms` | ❌ | ✅ | VM 관리 API |
| `/api/v1/quota` | ❌ | ✅ | 할당량 API |
| `/api/v1/pending` | ❌ | ✅ | 대기 목록 API |
| `/api/v1/users/*` | ❌ | ✅ | 사용자 승인/거부 API |

## 문제 해결

### nginx가 시작되지 않음

```bash
# 설정 문법 확인
sudo nginx -t

# 포트 사용 확인
sudo lsof -i :80
```

### API 서버 연결 실패

```bash
# API 서버 상태 확인
sudo systemctl status basphere-api

# 로그 확인
sudo journalctl -u basphere-api -f

# 포트 확인
sudo ss -tlnp | grep 8080
```

### 외부에서 내부 API 접근 시도

nginx 설정이 올바르게 적용되었다면 403 응답 반환:
```json
{"success":false,"message":"Access denied"}
```
