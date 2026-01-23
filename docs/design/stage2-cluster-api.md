# Stage 2: Kubernetes 클러스터 프로비저닝 (Cluster API)

## 개요

Stage 2에서는 Cluster API (CAPI)를 사용하여 사용자가 셀프서비스로 Kubernetes 클러스터를 생성할 수 있도록 합니다.

## 설계 원칙

- **Cluster API (CAPV)**: vSphere에서 K8s 클러스터 선언적 관리
- **Management 클러스터**: CAPI 컨트롤러 실행 환경
- **Ubuntu OVA**: Kubernetes SIG 제공 이미지 사용

## 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│              On-Premise Management 클러스터                   │
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │  Cluster API    │  │   Crossplane    │                   │
│  │  (CAPV+Ubuntu)  │  │   (vSphere)     │                   │
│  └────────┬────────┘  └─────────────────┘                   │
└───────────┼─────────────────────────────────────────────────┘
            │
            ▼ K8s 클러스터 생성
┌─────────────────────────────────────────────────────────────┐
│                     Bastion 서버                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  CLI 래퍼                                             │   │
│  │  - create-cluster → kubectl apply (CAPI manifest)    │   │
│  │  - delete-cluster / list-clusters / get-kubeconfig   │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│                  사용자 워크로드 클러스터                      │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  K8s 클러스터 (CAPI + Ubuntu OVA)                    │   │
│  │  Control Plane + Workers                            │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 클러스터 메타데이터 저장

```
/var/lib/basphere/clusters/{user}/{cluster-name}/
├── cluster.yaml         # CAPI manifest
├── kubeconfig           # 생성된 클러스터의 kubeconfig
└── metadata.json        # 클러스터 메타정보
```

## 클러스터 생성 흐름

1. 사용자가 `create-cluster` 실행
2. 래퍼가 IP 자동 할당 (Control Plane + Worker 수만큼)
3. CAPI manifest 자동 생성
4. `kubectl apply -f cluster.yaml` 실행 (Management 클러스터)
5. 클러스터 생성 완료 대기
6. kubeconfig 추출 및 저장
7. 결과 출력

## CLI 인터페이스

### create-cluster

```bash
$ create-cluster
? 클러스터 이름: my-cluster
? 클러스터 타입:
  [1] dev        - 1 Control Plane, 2 Workers
  [2] standard   - 3 Control Plane, 3 Workers
? 선택: 1
? Worker 노드 스펙:
  [1] small   - 2 vCPU, 4GB RAM
  [2] medium  - 4 vCPU, 8GB RAM
  [3] large   - 8 vCPU, 16GB RAM
? 선택: 2

✓ IP 할당: 10.254.0.72 ~ 10.254.0.74
✓ 클러스터 생성 요청 완료
✓ 프로비저닝 중...

진행 상황 확인: watch-cluster my-cluster
```

### 기타 명령어

```bash
list-clusters              # 내 클러스터 목록
get-kubeconfig <name>      # kubeconfig 출력
watch-cluster <name>       # 프로비저닝 상태 확인
delete-cluster <name>      # 클러스터 삭제
```

## 클러스터 스펙 정의

`/etc/basphere/specs.yaml`에 추가:

```yaml
cluster_specs:
  dev:
    control_plane_count: 1
    worker_count: 2
    control_plane_spec: medium
    worker_spec: medium
  standard:
    control_plane_count: 3
    worker_count: 3
    control_plane_spec: medium
    worker_spec: large
```

## Management 클러스터 설정

`/etc/basphere/config.yaml`에 추가:

```yaml
management_cluster:
  kubeconfig: /etc/basphere/management-kubeconfig
  namespace_prefix: user-

templates:
  kubernetes: ubuntu-2204-kube-v1.28.0
```

## 구현 산출물

| 항목 | 설명 |
|------|------|
| `create-cluster` | 클러스터 생성 CLI |
| `delete-cluster` | 클러스터 삭제 CLI |
| `list-clusters` | 클러스터 목록 CLI |
| `get-kubeconfig` | kubeconfig 조회 CLI |
| `watch-cluster` | 상태 확인 CLI |
| `cluster.yaml.tmpl` | CAPI manifest 템플릿 |

## 사전 요구사항

### Management 클러스터

- Kubernetes 클러스터 운영 중
- Cluster API 설치됨
- CAPV (Cluster API Provider vSphere) 설치됨
- Bastion에서 접근 가능한 kubeconfig

### VM 템플릿

- Kubernetes SIG Ubuntu OVA
  - CAPV 호환 이미지
  - https://github.com/kubernetes-sigs/image-builder 참조

## API 엔드포인트 (추가 예정)

| Method | 경로 | 설명 |
|--------|------|------|
| POST | `/api/v1/clusters` | 클러스터 생성 |
| GET | `/api/v1/clusters` | 클러스터 목록 |
| GET | `/api/v1/clusters/{name}` | 클러스터 상세 |
| DELETE | `/api/v1/clusters/{name}` | 클러스터 삭제 |
| GET | `/api/v1/clusters/{name}/kubeconfig` | kubeconfig 조회 |

## 관련 문서

- [로드맵](roadmap.md) - 전체 Stage 계획
- [아키텍처](architecture.md) - 전체 시스템 아키텍처
- [비전](vision.md) - 프로젝트 목표
