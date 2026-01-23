# Basphere 문서

이 디렉토리는 Basphere 프로젝트의 모든 문서를 체계적으로 관리합니다.

## 문서 구조

```
docs/
├── README.md                 # 이 파일 (문서 인덱스)
│
├── design/                   # IDP 설계 문서
│   ├── vision.md             # 프로젝트 비전 및 목표
│   ├── architecture.md       # 전체 아키텍처
│   ├── user-scenarios.md     # 사용자 시나리오
│   ├── infrastructure.md     # 온프레미스/OCI 인프라
│   └── roadmap.md            # Stage 1~3 상세 로드맵
│
├── operations/               # 운영 가이드
│   ├── installation.md       # 새 환경 설치 가이드
│   ├── troubleshooting.md    # 트러블슈팅
│   └── security.md           # 보안 설정 (SSH, fail2ban)
│
└── development/              # 개발 가이드
    └── contributing.md       # 개발 규칙, 코딩 스타일
```

## 문서 카테고리

### 설계 (design/)

IDP 전체 비전과 아키텍처 설계 문서입니다. 프로젝트의 방향성과 목표를 이해하는 데 필수입니다.

| 문서 | 설명 |
|------|------|
| [vision.md](design/vision.md) | 프로젝트 목표, 고려 기술, 플랫폼 구성 요소 |
| [architecture.md](design/architecture.md) | OCI/On-premise 아키텍처, 컨트롤 플레인 설계 |
| [user-scenarios.md](design/user-scenarios.md) | 사용자 워크플로우 (가입, 테넌트, 클러스터 생성) |
| [infrastructure.md](design/infrastructure.md) | 하드웨어 사양, 네트워크 구성 |
| [roadmap.md](design/roadmap.md) | Stage 1~3 상세 계획 및 진행 상태 |

### 운영 (operations/)

Basphere를 운영하고 관리하는 데 필요한 가이드입니다.

| 문서 | 설명 |
|------|------|
| [installation.md](operations/installation.md) | 새 환경에 Basphere 설치하기 |
| [troubleshooting.md](operations/troubleshooting.md) | 자주 발생하는 문제와 해결 방법 |
| [security.md](operations/security.md) | SSH 보안 강화, fail2ban 설정 |

### 개발 (development/)

Basphere 개발에 참여하기 위한 가이드입니다.

| 문서 | 설명 |
|------|------|
| [contributing.md](development/contributing.md) | 개발 규칙, 코딩 스타일, 커밋 메시지 |

## 컴포넌트별 문서

각 컴포넌트 디렉토리에도 해당 컴포넌트에 특화된 문서가 있습니다:

| 위치 | 설명 |
|------|------|
| [basphere-cli/README.md](../basphere-cli/README.md) | CLI 설치 및 운영자 가이드 |
| [basphere-cli/docs/user-guide.md](../basphere-cli/docs/user-guide.md) | 사용자 가이드 |
| [basphere-api/README.md](../basphere-api/README.md) | API 서버 개요 및 엔드포인트 |
| [deploy/README.md](../deploy/README.md) | 배포 설정 (nginx, systemd) |

## 빠른 참조

### 새 환경에 배포하려면?
→ [operations/installation.md](operations/installation.md)

### 문제가 발생했다면?
→ [operations/troubleshooting.md](operations/troubleshooting.md)

### IDP 전체 비전을 이해하려면?
→ [design/vision.md](design/vision.md) → [design/architecture.md](design/architecture.md)

### 코드를 수정하려면?
→ [development/contributing.md](development/contributing.md)
