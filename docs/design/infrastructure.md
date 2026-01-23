# Basphere 인프라 사양

홈랩 환경의 하드웨어 및 네트워크 사양입니다.

## On-premise 서버

### 컴퓨트 서버 (ESXi 호스트)

| 항목 | 값 |
|------|-----|
| **모델** | Beelink SER5 Max |
| **CPU** | AMD Ryzen 7 5800H (8코어 16스레드) |
| **RAM** | 64GB DDR4 3200MHz |
| **스토리지** | 256GB NVMe SSD |
| **네트워크** | 1Gbps 이더넷 2개 |
| **OS** | VMware ESXi 8.0u3 |
| **수량** | 6대 |

### 총 컴퓨트 리소스

| 자원 | 값 |
|------|-----|
| 총 vCPU | 96 vCPU (6 x 16스레드) |
| 총 메모리 | 384GB (6 x 64GB) |

## 네트워크

### 최상단 라우터

| 항목 | 값 |
|------|-----|
| **모델** | Topton mini pc |
| **CPU** | Intel N100 (4코어 4스레드) |
| **RAM** | 8GB DDR5 4800MHz |
| **스토리지** | 128GB NVMe SSD |
| **네트워크** | 2.5Gbps 이더넷 6개 (Intel I226-V) |
| **OS** | OPNsense 25.7.10-amd64 |
| **수량** | 1대 |

### L2 스위치

| 항목 | 값 |
|------|-----|
| **모델** | TP-LINK SG2218 |
| **RJ45 포트** | 1Gbps 16포트 |
| **SFP 포트** | 1Gbps 2포트 |
| **수량** | 1대 |

## 스토리지

### NAS

| 항목 | 값 |
|------|-----|
| **모델** | Synology DS1515+ |
| **CPU** | Intel Atom C2538 (4코어 4스레드) |
| **RAM** | 16GB DDR3 |
| **네트워크** | 1Gbps 이더넷 4포트 |
| **OS** | DSM 7.1.1-42962 Update 9 |

### 스토리지 구성

| 타입 | 모델 | 수량 | RAID |
|------|------|------|------|
| **SSD** | Samsung SSD 870 EVO 4TB | 3개 | RAID 5 |
| **HDD** | Western Digital 4TB | 2개 | RAID 1 |

### 가용 용량

| 풀 | 용량 |
|-----|------|
| SSD (RAID 5) | ~8TB |
| HDD (RAID 1) | ~4TB |

## 가상화 플랫폼

| 구성 요소 | 버전 |
|----------|------|
| **ESXi** | VMware ESXi 8.0u3 |
| **vCenter** | VMware vCenter Server 8.0u3 |

## OCI (Oracle Cloud Infrastructure)

| 항목 | 값 |
|------|-----|
| **리전** | 싱가포르 1 |
| **용도** | Talos Omni, 모니터링 허브 |
| **연결** | Site-to-Site VPN (On-premise 연결) |

## 네트워크 토폴로지

```
인터넷
    │
    ▼
OCI (싱가포르)
    │ Site-to-Site VPN
    ▼
┌─────────────────────────────────────────┐
│          최상단 라우터 (OPNsense)        │
│          Topton mini pc                 │
│          2.5Gbps x 6                    │
└─────────────────┬───────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│          L2 스위치 (TP-LINK)            │
│          1Gbps x 16                     │
└────┬────────────────────────────────┬───┘
     │                                │
     ▼                                ▼
┌─────────────┐              ┌─────────────┐
│ ESXi Host 1 │  ...         │ ESXi Host 6 │
│ (Beelink)   │              │ (Beelink)   │
└─────────────┘              └─────────────┘
                  │
                  ▼
         ┌─────────────┐
         │  Synology   │
         │    NAS      │
         └─────────────┘
```

## 관련 문서

- [아키텍처](architecture.md) - 전체 시스템 아키텍처
- [비전](vision.md) - 프로젝트 목표
