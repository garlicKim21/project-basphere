# Stage 2 구현 계획: Cluster API 기반 Kubernetes 프로비저닝

## 개요

Stage 2에서는 Cluster API (CAPI)와 CAPV (Cluster API Provider vSphere)를 사용하여 Kubernetes 클러스터를 프로비저닝합니다.

**핵심 결정사항:**
- Cluster API (CAPV) 사용 (Terraform + kubeadm 방식 아님)
- Management Cluster: kind로 시작
- 네트워크: 현재 10.254.0.0/21 공간 유지
- 테넌트 격리: Stage 2 범위 아님 (이후 단계)

## 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│                         Bastion 서버                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   CLI       │  │  API Server │  │   Management Cluster    │  │
│  │  (Bash)     │──│   (Go)      │──│   (kind)                │  │
│  └─────────────┘  └─────────────┘  │  ┌─────────────────────┐│  │
│                                    │  │  CAPI Controller    ││  │
│                                    │  │  CAPV Controller    ││  │
│                                    │  └─────────────────────┘│  │
│                                    └───────────┬─────────────┘  │
└────────────────────────────────────────────────┼────────────────┘
                                                 │ kubectl apply
                                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      vSphere (vCenter)                           │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              사용자 워크로드 클러스터                         ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         ││
│  │  │ Control     │  │   Worker    │  │   Worker    │  ...    ││
│  │  │ Plane VM    │  │   VM        │  │   VM        │         ││
│  │  │ (Ubuntu)    │  │   (Ubuntu)  │  │   (Ubuntu)  │         ││
│  │  └─────────────┘  └─────────────┘  └─────────────┘         ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## Phase 1: Management Cluster 구축

### 1.1 사전 요구사항

Bastion 서버에 설치해야 할 도구:

| 도구 | 용도 | 설치 방법 |
|------|------|----------|
| Docker | kind 컨테이너 런타임 | apt install docker.io |
| kind | Management Cluster | GitHub releases |
| kubectl | Kubernetes CLI | apt install kubectl |
| clusterctl | Cluster API CLI | GitHub releases |

### 1.2 kind 클러스터 생성

```bash
# /etc/basphere/kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: basphere-mgmt
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 6443
        hostPort: 6443
        protocol: TCP
networking:
  podSubnet: "10.244.0.0/16"
  serviceSubnet: "10.96.0.0/12"
```

```bash
# Management Cluster 생성
kind create cluster --config /etc/basphere/kind-config.yaml

# kubeconfig 저장
kind get kubeconfig --name basphere-mgmt > /etc/basphere/management-kubeconfig
chmod 600 /etc/basphere/management-kubeconfig
```

### 1.3 Cluster API 설치

```bash
# clusterctl 설치
curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.6.0/clusterctl-linux-amd64 -o clusterctl
chmod +x clusterctl
sudo mv clusterctl /usr/local/bin/

# CAPV 초기화
export VSPHERE_USERNAME="administrator@vsphere.local"
export VSPHERE_PASSWORD="password"

clusterctl init \
  --infrastructure vsphere \
  --kubeconfig /etc/basphere/management-kubeconfig
```

### 1.4 vSphere 인증 설정

```bash
# /etc/basphere/capv-credentials.yaml
apiVersion: v1
kind: Secret
metadata:
  name: capv-credentials
  namespace: capv-system
type: Opaque
stringData:
  username: "${VSPHERE_USER}"
  password: "${VSPHERE_PASSWORD}"
```

## Phase 2: CAPI Manifest 템플릿

### 2.1 클러스터 템플릿 구조

```
basphere-cli/templates/capi/
├── cluster.yaml.tmpl           # 클러스터 정의
├── control-plane.yaml.tmpl     # Control Plane 정의
├── machine-deployment.yaml.tmpl # Worker 노드 정의
└── cloud-init/
    ├── control-plane.yaml.tmpl # CP cloud-init
    └── worker.yaml.tmpl        # Worker cloud-init
```

### 2.2 클러스터 템플릿 예시

```yaml
# cluster.yaml.tmpl
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: ${CLUSTER_NAME}
  namespace: ${NAMESPACE}
  labels:
    owner: ${OWNER}
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
        - 192.168.0.0/16
    services:
      cidrBlocks:
        - 10.128.0.0/12
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: KubeadmControlPlane
    name: ${CLUSTER_NAME}-control-plane
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: VSphereCluster
    name: ${CLUSTER_NAME}
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: VSphereCluster
metadata:
  name: ${CLUSTER_NAME}
  namespace: ${NAMESPACE}
spec:
  controlPlaneEndpoint:
    host: ${CONTROL_PLANE_IP}
    port: 6443
  identityRef:
    kind: Secret
    name: capv-credentials
  server: ${VSPHERE_SERVER}
  thumbprint: ${VSPHERE_TLS_THUMBPRINT}
```

### 2.3 Control Plane 템플릿

```yaml
# control-plane.yaml.tmpl
---
apiVersion: controlplane.cluster.x-k8s.io/v1beta1
kind: KubeadmControlPlane
metadata:
  name: ${CLUSTER_NAME}-control-plane
  namespace: ${NAMESPACE}
spec:
  kubeadmConfigSpec:
    clusterConfiguration:
      apiServer:
        extraArgs:
          cloud-provider: external
      controllerManager:
        extraArgs:
          cloud-provider: external
    initConfiguration:
      nodeRegistration:
        criSocket: /var/run/containerd/containerd.sock
        kubeletExtraArgs:
          cloud-provider: external
        name: '{{ local_hostname }}'
    joinConfiguration:
      nodeRegistration:
        criSocket: /var/run/containerd/containerd.sock
        kubeletExtraArgs:
          cloud-provider: external
        name: '{{ local_hostname }}'
  machineTemplate:
    infrastructureRef:
      apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
      kind: VSphereMachineTemplate
      name: ${CLUSTER_NAME}-control-plane
  replicas: ${CONTROL_PLANE_COUNT}
  version: ${KUBERNETES_VERSION}
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: VSphereMachineTemplate
metadata:
  name: ${CLUSTER_NAME}-control-plane
  namespace: ${NAMESPACE}
spec:
  template:
    spec:
      cloneMode: linkedClone
      datacenter: ${VSPHERE_DATACENTER}
      datastore: ${VSPHERE_DATASTORE}
      diskGiB: ${CONTROL_PLANE_DISK}
      folder: ${VSPHERE_FOLDER}
      memoryMiB: ${CONTROL_PLANE_MEMORY}
      network:
        devices:
          - dhcp4: false
            networkName: ${VSPHERE_NETWORK}
      numCPUs: ${CONTROL_PLANE_CPU}
      resourcePool: ${VSPHERE_RESOURCE_POOL}
      server: ${VSPHERE_SERVER}
      storagePolicyName: ""
      template: ${KUBERNETES_TEMPLATE}
      thumbprint: ${VSPHERE_TLS_THUMBPRINT}
```

### 2.4 Worker 노드 템플릿

```yaml
# machine-deployment.yaml.tmpl
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: ${CLUSTER_NAME}-workers
  namespace: ${NAMESPACE}
spec:
  clusterName: ${CLUSTER_NAME}
  replicas: ${WORKER_COUNT}
  selector:
    matchLabels: null
  template:
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: KubeadmConfigTemplate
          name: ${CLUSTER_NAME}-workers
      clusterName: ${CLUSTER_NAME}
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
        kind: VSphereMachineTemplate
        name: ${CLUSTER_NAME}-workers
      version: ${KUBERNETES_VERSION}
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: VSphereMachineTemplate
metadata:
  name: ${CLUSTER_NAME}-workers
  namespace: ${NAMESPACE}
spec:
  template:
    spec:
      cloneMode: linkedClone
      datacenter: ${VSPHERE_DATACENTER}
      datastore: ${VSPHERE_DATASTORE}
      diskGiB: ${WORKER_DISK}
      folder: ${VSPHERE_FOLDER}
      memoryMiB: ${WORKER_MEMORY}
      network:
        devices:
          - dhcp4: false
            networkName: ${VSPHERE_NETWORK}
      numCPUs: ${WORKER_CPU}
      resourcePool: ${VSPHERE_RESOURCE_POOL}
      server: ${VSPHERE_SERVER}
      template: ${KUBERNETES_TEMPLATE}
      thumbprint: ${VSPHERE_TLS_THUMBPRINT}
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: KubeadmConfigTemplate
metadata:
  name: ${CLUSTER_NAME}-workers
  namespace: ${NAMESPACE}
spec:
  template:
    spec:
      joinConfiguration:
        nodeRegistration:
          criSocket: /var/run/containerd/containerd.sock
          kubeletExtraArgs:
            cloud-provider: external
          name: '{{ local_hostname }}'
```

## Phase 3: CLI 구현

### 3.1 명령어 목록

| 명령어 | 설명 | 구현 방식 |
|--------|------|----------|
| create-cluster | 클러스터 생성 | kubectl apply (CAPI manifest) |
| delete-cluster | 클러스터 삭제 | kubectl delete |
| list-clusters | 클러스터 목록 | kubectl get clusters |
| get-kubeconfig | kubeconfig 조회 | clusterctl get kubeconfig |
| watch-cluster | 상태 모니터링 | kubectl get cluster -w |

### 3.2 create-cluster 흐름

```bash
# 사용자 실행
$ create-cluster

# 1. 대화형 입력
? 클러스터 이름: my-cluster
? 클러스터 타입:
  [1] dev        - 1 Control Plane, 2 Workers (기본)
  [2] standard   - 3 Control Plane, 3 Workers
? 선택: 1
? Worker 노드 스펙:
  [1] small   - 2 vCPU, 4GB RAM
  [2] medium  - 4 vCPU, 8GB RAM (기본)
  [3] large   - 8 vCPU, 16GB RAM
? 선택: 2

# 2. IP 할당 (IPAM)
✓ Control Plane IP: 10.254.0.72
✓ Worker IPs: 10.254.0.73, 10.254.0.74

# 3. CAPI manifest 생성
✓ 클러스터 manifest 생성: /var/lib/basphere/clusters/user/my-cluster/cluster.yaml

# 4. kubectl apply 실행 (Management 클러스터에)
✓ 클러스터 생성 요청 완료

# 5. 프로비저닝 상태 확인 안내
진행 상황 확인: watch-cluster my-cluster
```

### 3.3 스크립트 구조

```bash
# basphere-cli/scripts/user/create-cluster
#!/bin/bash
set -euo pipefail

source /usr/local/lib/basphere/common.sh
source /usr/local/lib/basphere/cluster-common.sh

# 1. 인자 파싱
parse_cluster_args "$@"

# 2. 대화형 입력 (필요시)
if [[ -z "${CLUSTER_NAME:-}" ]]; then
    prompt_cluster_name
fi
if [[ -z "${CLUSTER_TYPE:-}" ]]; then
    prompt_cluster_type
fi

# 3. 검증
validate_resource_name "$CLUSTER_NAME"
validate_cluster_quota

# 4. API 호출 또는 직접 실행
if [[ "${API_MODE:-false}" == "true" ]]; then
    # API 서버 호출 (Provisioner가 처리)
    api_call "POST" "/api/v1/clusters" "{
        \"name\": \"$CLUSTER_NAME\",
        \"type\": \"$CLUSTER_TYPE\",
        \"worker_spec\": \"$WORKER_SPEC\"
    }"
else
    # 직접 실행
    create_cluster_direct
fi
```

## Phase 4: API 서버 확장

### 4.1 새로운 엔드포인트

```go
// router 등록
r.Route("/api/v1/clusters", func(r chi.Router) {
    r.Post("/", h.apiCreateCluster)
    r.Get("/", h.apiListClusters)
    r.Get("/{name}", h.apiGetCluster)
    r.Delete("/{name}", h.apiDeleteCluster)
    r.Get("/{name}/kubeconfig", h.apiGetKubeconfig)
    r.Get("/{name}/status", h.apiGetClusterStatus)
    r.Get("/quota", h.apiGetClusterQuota)
})
```

### 4.2 데이터 모델

```go
// model/cluster.go
type ClusterStatus string

const (
    ClusterStatusPending      ClusterStatus = "pending"
    ClusterStatusProvisioning ClusterStatus = "provisioning"
    ClusterStatusReady        ClusterStatus = "ready"
    ClusterStatusDeleting     ClusterStatus = "deleting"
    ClusterStatusFailed       ClusterStatus = "failed"
)

type Cluster struct {
    Name              string        `json:"name"`
    Owner             string        `json:"owner"`
    Type              string        `json:"type"`          // dev, standard
    K8sVersion        string        `json:"k8s_version"`
    ControlPlaneCount int           `json:"control_plane_count"`
    WorkerCount       int           `json:"worker_count"`
    WorkerSpec        string        `json:"worker_spec"`   // small, medium, large
    ControlPlaneIP    string        `json:"control_plane_ip"`
    WorkerIPs         []string      `json:"worker_ips"`
    Status            ClusterStatus `json:"status"`
    CreatedAt         time.Time     `json:"created_at"`
    ReadyAt           *time.Time    `json:"ready_at,omitempty"`
    KubeconfigPath    string        `json:"kubeconfig_path,omitempty"`
}

type CreateClusterInput struct {
    Name       string `json:"name"`
    Type       string `json:"type"`        // dev, standard
    WorkerSpec string `json:"worker_spec"` // small, medium, large
}
```

### 4.3 Provisioner 인터페이스 확장

```go
// provisioner/provisioner.go
type Provisioner interface {
    // 기존 VM 메서드
    CreateVM(username string, input *model.CreateVMInput) (*model.VM, error)
    DeleteVM(username, vmName string) error
    ListVMs(username string) ([]*model.VM, error)
    // ...

    // 클러스터 메서드 추가
    CreateCluster(username string, input *model.CreateClusterInput) (*model.Cluster, error)
    DeleteCluster(username, clusterName string) error
    ListClusters(username string) ([]*model.Cluster, error)
    GetCluster(username, clusterName string) (*model.Cluster, error)
    GetKubeconfig(username, clusterName string) ([]byte, error)
    ClusterExists(username, clusterName string) bool
    GetClusterQuota(username string) (*model.ClusterQuota, error)
}
```

## Phase 5: 설정 파일 확장

### 5.1 config.yaml 확장

```yaml
# /etc/basphere/config.yaml 에 추가

management_cluster:
  kubeconfig: /etc/basphere/management-kubeconfig
  namespace_prefix: user-  # 사용자별 네임스페이스 (user-john, user-alice)

templates:
  kubernetes: ubuntu-2204-kube-v1.28.0  # CAPV 호환 이미지
```

### 5.2 specs.yaml 확장

```yaml
# /etc/basphere/specs.yaml 에 추가

cluster_types:
  dev:
    description: "Development cluster"
    control_plane_count: 1
    worker_count: 2
    control_plane_spec: medium
    worker_spec_default: medium
  standard:
    description: "Standard cluster (HA)"
    control_plane_count: 3
    worker_count: 3
    control_plane_spec: medium
    worker_spec_default: large

cluster_node_specs:
  small:
    cpu: 2
    memory_mb: 4096
    disk_gb: 50
  medium:
    cpu: 4
    memory_mb: 8192
    disk_gb: 100
  large:
    cpu: 8
    memory_mb: 16384
    disk_gb: 200

cluster_quotas:
  default:
    max_clusters: 3
    max_nodes_per_cluster: 10
```

## Phase 6: 데이터 저장 구조

```
/var/lib/basphere/
├── clusters/
│   └── {user}/
│       └── {cluster-name}/
│           ├── cluster.yaml       # 생성된 CAPI manifest
│           ├── kubeconfig         # 워크로드 클러스터 kubeconfig
│           └── metadata.json      # 클러스터 메타데이터
├── ipam/
│   ├── allocations.tsv           # 사용자별 IP 블록
│   └── leases.tsv                # VM/클러스터 IP 사용 현황
```

## 구현 일정

| Phase | 작업 | 예상 기간 |
|-------|------|----------|
| Phase 1 | Management Cluster 구축 | 2-3일 |
| Phase 2 | CAPI Manifest 템플릿 | 3-4일 |
| Phase 3 | CLI 구현 | 3-4일 |
| Phase 4 | API 서버 확장 | 3-4일 |
| Phase 5 | 설정 파일 확장 | 1일 |
| Phase 6 | 테스트 및 문서화 | 2-3일 |
| **Total** | | **2-3주** |

## 사전 준비 사항

### vSphere 환경

1. **CAPV 호환 VM 템플릿**
   - Ubuntu 22.04 + Kubernetes components 설치
   - https://github.com/kubernetes-sigs/image-builder 사용
   - 또는 OVA 이미지 다운로드

2. **네트워크 설정**
   - DHCP 비활성화 (정적 IP 사용)
   - 기존 10.254.0.0/21 네트워크 사용

3. **vSphere 권한**
   - VM 생성/삭제 권한
   - 폴더 생성 권한
   - 리소스 풀 접근 권한

### Bastion 서버

1. **Docker 설치**
   ```bash
   sudo apt install -y docker.io
   sudo usermod -aG docker basphere
   ```

2. **kind, kubectl, clusterctl 설치**
   - 설치 스크립트에 추가

## 다음 단계

Phase 1 (Management Cluster 구축)부터 시작:

1. Docker 설치 스크립트 작성
2. kind 설치 및 설정 스크립트 작성
3. CAPV 초기화 스크립트 작성
4. Management Cluster 상태 확인 명령어 작성

## 관련 문서

- [Stage 2 설계](stage2-cluster-api.md) - 기본 설계
- [로드맵](roadmap.md) - 전체 Stage 계획
- [아키텍처](architecture.md) - 전체 시스템 아키텍처
